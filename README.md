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


