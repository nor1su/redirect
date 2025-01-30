# Redirector

A simple HTTP redirect server that redirects incoming requests to a specified destination while tracking redirect statistics. It also includes an optional URL filtering mechanism to allow or block redirects based on specific keywords.

## Build

```sh
go build -o redirector main.go
```

## Usage

Run the redirector with the desired base URL, address, and optional filtering.

```sh
./redirector -base="https://example.com" -addr=":8080" -filter="keyword1,keyword2" -filter-count=5
```

### Available Flags

- `-base` *(required)*: Destination URL for redirection.
- `-addr` *(optional)*: Address and port to listen on (default `:8080`).
- `-filter` *(optional)*: Comma-separated list of keywords to filter which URLs should be redirected. If a URL contains any of these words, it will be redirected. Otherwise, a `403 Forbidden` response will be returned.
- `-filter-count` *(optional)*: Maximum number of keywords to consider (default `0`, meaning no limit).

### Examples

Redirect all requests to `https://example.com`:
```sh
./redirector -base="https://example.com"
```

Redirect only URLs containing `promo` or `offer`:
```sh
./redirector -base="https://example.com" -filter="promo,offer"
```

Limit the filter to a maximum of 3 keywords:
```sh
./redirector -base="https://example.com" -filter="sale,discount,deal,special" -filter-count=3
```

## Statistics

The server tracks the number of redirects and provides statistics via HTTP:

- **HTML Statistics Page:**  
  Accessible at:  
  ```
  http://localhost:8080/<random-path>
  ```
- **JSON Statistics API:**  
  Accessible at:  
  ```
  http://localhost:8080/<random-json-path>
  ```
- **Reset Statistics (POST request required):**  
  ```
  http://localhost:8080/<random-reset-path>
  ```

The exact paths are randomly generated and stored in `paths.json` upon startup.

## Nginx Reverse Proxy Configuration

To run the redirector behind an Nginx reverse proxy, use the following configuration:

```nginx
location / {
    proxy_pass http://localhost:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

This setup allows Nginx to forward all incoming requests to the redirector while preserving client information.

## Notes

- The server maintains persistent statistics in `stats.json`.
- Randomized paths for statistics and reset functions are stored in `paths.json`.
- Ensure `stats.json` and `paths.json` are writable for persistence across restarts.
- The filter feature is optional and defaults to allowing all requests.

---

This tool is useful for tracking and controlling redirects with optional keyword filtering.
