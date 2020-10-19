# HTTP Cache

HTTP cache implements a resource based http server that please can use as a cache. The cache supports storing files
via PUT requests and retrieving them again through GET requests. Really any http server (e.g. nginx) can be used as a 
cache for please however this is a lightweight and easy to configure option.

## Usage

  http_cache [OPTIONS]

HTTP Cache options:
  -v, --verbosity= Verbosity of output (higher number = more output) (default: warning)
  -d, --dir=       The directory to store cached artifacts in.
  -p, --port=      The port to run the server on
