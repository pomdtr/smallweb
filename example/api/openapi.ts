export default {
  "components": {
    "schemas": {
      "ErrorDetail": {
        "additionalProperties": false,
        "properties": {
          "location": {
            "description": "Where the error occurred, e.g. 'body.items[3].tags' or 'path.thing-id'",
            "type": "string"
          },
          "message": {
            "description": "Error message text",
            "type": "string"
          },
          "value": {
            "description": "The value at the given location"
          }
        },
        "type": "object"
      },
      "ErrorModel": {
        "additionalProperties": false,
        "properties": {
          "$schema": {
            "description": "A URL to the JSON Schema for this object.",
            "examples": [
              "https://example.com/schemas/ErrorModel.json"
            ],
            "format": "uri",
            "readOnly": true,
            "type": "string"
          },
          "detail": {
            "description": "A human-readable explanation specific to this occurrence of the problem.",
            "examples": [
              "Property foo is required but is missing."
            ],
            "type": "string"
          },
          "errors": {
            "description": "Optional list of individual error details",
            "items": {
              "$ref": "#/components/schemas/ErrorDetail"
            },
            "type": [
              "array",
              "null"
            ]
          },
          "instance": {
            "description": "A URI reference that identifies the specific occurrence of the problem.",
            "examples": [
              "https://example.com/error-log/abc123"
            ],
            "format": "uri",
            "type": "string"
          },
          "status": {
            "description": "HTTP status code",
            "examples": [
              400
            ],
            "format": "int64",
            "type": "integer"
          },
          "title": {
            "description": "A short, human-readable summary of the problem type. This value should not change between occurrences of the error.",
            "examples": [
              "Bad Request"
            ],
            "type": "string"
          },
          "type": {
            "default": "about:blank",
            "description": "A URI reference to human-readable documentation for the error.",
            "examples": [
              "https://example.com/errors/example"
            ],
            "format": "uri",
            "type": "string"
          }
        },
        "type": "object"
      },
      "GetAppOutputBody": {
        "additionalProperties": false,
        "properties": {
          "$schema": {
            "description": "A URL to the JSON Schema for this object.",
            "examples": [
              "https://example.com/schemas/GetAppOutputBody.json"
            ],
            "format": "uri",
            "readOnly": true,
            "type": "string"
          }
        },
        "type": "object"
      },
      "GetAppsOutputBody": {
        "additionalProperties": false,
        "properties": {
          "$schema": {
            "description": "A URL to the JSON Schema for this object.",
            "examples": [
              "https://example.com/schemas/GetAppsOutputBody.json"
            ],
            "format": "uri",
            "readOnly": true,
            "type": "string"
          },
          "apps": {
            "items": {
              "type": "string"
            },
            "type": [
              "array",
              "null"
            ]
          }
        },
        "required": [
          "apps"
        ],
        "type": "object"
      }
    }
  },
  "info": {
    "title": "My API",
    "version": "1.0.0"
  },
  "openapi": "3.1.0",
  "paths": {
    "/v1/apps": {
      "get": {
        "responses": {
          "200": {
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/GetAppsOutputBody"
                }
              }
            },
            "description": "OK"
          },
          "default": {
            "content": {
              "application/problem+json": {
                "schema": {
                  "$ref": "#/components/schemas/ErrorModel"
                }
              }
            },
            "description": "Error"
          }
        }
      }
    },
    "/v1/apps/{app}": {
      "get": {
        "responses": {
          "200": {
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/GetAppOutputBody"
                }
              }
            },
            "description": "OK"
          },
          "default": {
            "content": {
              "application/problem+json": {
                "schema": {
                  "$ref": "#/components/schemas/ErrorModel"
                }
              }
            },
            "description": "Error"
          }
        }
      }
    },
    "/v1/blobs": {
      "get": {
        "description": "List Blobs",
        "parameters": [
          {
            "description": "Filter blobs by prefix",
            "explode": false,
            "in": "query",
            "name": "prefix",
            "schema": {
              "description": "Filter blobs by prefix",
              "type": "string"
            }
          }
        ],
        "responses": {
          "200": {
            "content": {
              "application/json": {
                "schema": {
                  "items": {
                    "type": "string"
                  },
                  "type": [
                    "array",
                    "null"
                  ]
                }
              }
            },
            "description": "OK"
          },
          "default": {
            "content": {
              "application/problem+json": {
                "schema": {
                  "$ref": "#/components/schemas/ErrorModel"
                }
              }
            },
            "description": "Error"
          }
        }
      }
    },
    "/v1/blobs/{key}": {
      "get": {
        "description": "Retrieve a blob by its key",
        "operationId": "getBlob",
        "parameters": [
          {
            "description": "The blob key",
            "in": "path",
            "name": "key",
            "required": true,
            "schema": {
              "description": "The blob key",
              "type": "string"
            }
          }
        ],
        "responses": {
          "200": {
            "content": {
              "application/octet-stream": {}
            },
            "description": "Blob Content",
            "headers": {
              "Content-Type": {
                "schema": {
                  "type": "string"
                }
              }
            }
          },
          "default": {
            "content": {
              "application/problem+json": {
                "schema": {
                  "$ref": "#/components/schemas/ErrorModel"
                }
              }
            },
            "description": "Error"
          }
        }
      }
    },
    "/v1/email": {
      "post": {
        "responses": {
          "204": {
            "description": "No Content"
          },
          "default": {
            "content": {
              "application/problem+json": {
                "schema": {
                  "$ref": "#/components/schemas/ErrorModel"
                }
              }
            },
            "description": "Error"
          }
        }
      }
    },
    "/v1/sqlite/batch": {
      "post": {
        "responses": {
          "204": {
            "description": "No Content"
          },
          "default": {
            "content": {
              "application/problem+json": {
                "schema": {
                  "$ref": "#/components/schemas/ErrorModel"
                }
              }
            },
            "description": "Error"
          }
        }
      }
    },
    "/v1/sqlite/query": {
      "post": {
        "responses": {
          "204": {
            "description": "No Content"
          },
          "default": {
            "content": {
              "application/problem+json": {
                "schema": {
                  "$ref": "#/components/schemas/ErrorModel"
                }
              }
            },
            "description": "Error"
          }
        }
      }
    }
  }
} as const;
