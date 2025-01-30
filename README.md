```markdown
# Redirector

A simple HTTP redirect server. It redirects incoming requests to a specified destination and tracks redirect statistics.

### Build

    go build -o redirector main.go

## Usage

Run the redirector with the desired base URL and address.

    ./redirector -base="https://example.com" -addr=":8080"

- `-base`: Destination URL for redirection.
- `-addr`: Address and port to listen on (default `:8080`).


    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
