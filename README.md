```markdown
# Redirector

A simple HTTP redirect server. It redirects incoming requests to a specified destination and tracks redirect statistics.

### Build

1. Clone the repository or download the source code.
2. Navigate to the project directory.
3. Build the executable:


    go build -o redirector main.go


## Usage

Run the redirector with the desired base URL and address.

    ./redirector -base="https://example.com" -addr=":8080"


- `-base`: Destination URL for redirection.
- `-addr`: Address and port to listen on (default `:8080`).


