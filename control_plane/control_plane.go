package control_plane

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
)

var (
	// key be like username:password->redis_url
	backendAddrTable = make(map[string]string)
	tableMutex       = &sync.RWMutex{}
	controlPlaneURL  = "http://localhost:8080"
)
var auth_token = os.Getenv("AUTH_TOKEN")

// Call control plane API for user auth
func GetBackendAddress(username, password string) (string, error) {
	url := fmt.Sprintf("%s/api/v1/redis/route-table?username=%s&password=%s&auth_token=%s", controlPlaneURL, username, password, auth_token)
	key := fmt.Sprintf("%s:%s", username, password)
	tableMutex.RLock()
	addr, ok := backendAddrTable[key]
	tableMutex.RUnlock()
	if ok {
		return addr, nil
	}
	response, err := http.Get(url)
	if err != nil {
		log.Println("control plane error:", err)
		return "", err
	}
	defer response.Body.Close()
	var result struct {
		Message string `json:"message"`
		Success bool   `json:"success"`
		Backend string `json:"backend_url"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", err
	}
	if !result.Success {
		return "", fmt.Errorf("invalid credentials")
	}
	tableMutex.Lock()
	backendAddrTable[key] = result.Backend
	tableMutex.Unlock()
	return result.Backend, nil
}

func StartUpdateServer() {
	http.HandleFunc("/api/v1/redis/update-table", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Update request received")
		var req struct {
			Message   string `json:"message"`
			Success   bool   `json:"success"`
			AuthToken string `json:"auth_token"`
			OldKey    string `json:"old_key"`
			NewKey    string `json:"new_key"`
			Backend   string `json:"backend_url"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
		if req.AuthToken != auth_token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		tableMutex.Lock()
		defer tableMutex.Unlock()

		if req.OldKey != "" {
			if _, exists := backendAddrTable[req.OldKey]; exists {
				delete(backendAddrTable, req.OldKey)
				fmt.Println("Deleted old key:", req.OldKey)
			}
		}
		backendAddrTable[req.NewKey] = req.Backend
		fmt.Println("Updated mapping:", req.NewKey, "->", req.Backend)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	go func() {
		fmt.Println("Update server listening on :9000")
		if err := http.ListenAndServe(":9000", nil); err != nil {
			fmt.Println("Update server error:", err)
		}
	}()
}
