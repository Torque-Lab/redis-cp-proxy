# Redis TCP Gateway

A high-performance TCP proxy for Redis that handles SSL/TLS termination, authentication, and acts as a secure relay between clients and Redis servers.

## Features

- SSL/TLS termination for secure communication
-  Authentication layer for client connections
-  High-performance TCP relay with minimal latency
-  Bidirectional data streaming
-  Dynamic Routing
## Architecture

sequenceDiagram
    participant C as Client
    participant P as Proxy
    participant R as Redis Server

    %% Connection Phase
    C->>P: TCP Connection
    P->>C: Connection Established
    
    %% SSL Handshake (Mandatory)
    C->>P: SSL Handshake
    P->>C: SSL Handshake Complete
    
    %% Authentication
    C->>P: AUTH <credentials>
    P-->>C: +OK (if authentication valid)
    
    %% TCP Relay Phase
    P->>R: Redis Commands
    C<<->>R:  Normal Redis Operation
 





## License

[MIT](LICENSE)

## Contributing
Contributions are welcome! 

