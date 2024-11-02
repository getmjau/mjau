package mjau

var InitSampleConfig = `# This is a sample mjau configuration file
# You can use this file as a template for your own configuration
environments:
  - name: default
    variables:
      - key: host
        value: httpbin.org
      - key: uuid
        value: "{{$uuid()}}"
requests:
  - name: test
    pre_commands:
      - command: add_variable
        description: "add variable test"
        variable: test99
        value: "test99"
    url: http://{{environment.host}}/get
    method: GET
    headers:
      - key: Content-Type
        value: application/json
    commands:
      - command: add_variable
        description: "add variable test"
        variable: test
        value: "test:\n{{environment.host}}\n{{$timestamp()}}\n{{$random(100)}}\n{{$uuid()}}"
      - command: add_json_variable
        description: "add origin from response.body to test2"
        variable: test2
        from_variable: response.body
        path: origin

  - name: healthz
    url: http://{{environment.host}}/get
    method: GET
    headers:
      - key: Content-Type
        value: application/json
    commands:
      - command: echo
        description: "print response code"
        value: "response code: {{response.status_code}}"
    asserts:
      - description: "status code is 200"
        variable: response.status_code
        comparison: "=="
        value: 200
      - description: "status code is less than 400"
        variable: response.status_code
        comparison: "<="
        value: 400
      - description: "content type is application/json"
        variable: response.headers.Content-Type
        comparison: "=="
        value: application/json
      - description: "body contains origin"
        variable: response.body
        comparison: contains
        value: "origin"
`
