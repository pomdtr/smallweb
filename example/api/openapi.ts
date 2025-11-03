export default {
    "components": {
        "schemas": {
            "BatchRequest": {
                "properties": {
                    "mode": { "enum": ["read", "write", "deferred"], "type": "string" },
                    "statements": {
                        "items": {
                            "oneOf": [{ "type": "string" }, {
                                "$ref": "#/components/schemas/StatementObject",
                            }],
                        },
                        "type": "array",
                    },
                },
                "required": ["statements"],
                "type": "object",
            },
            "ExecuteRequest": {
                "properties": {
                    "statement": {
                        "oneOf": [{
                            "description": "Simple SQL statement string",
                            "type": "string",
                        }, { "$ref": "#/components/schemas/StatementObject" }],
                    },
                },
                "required": ["statement"],
                "type": "object",
            },
            "ExecuteResult": {
                "properties": {
                    "columnTypes": { "items": { "type": "string" }, "type": "array" },
                    "columns": { "items": { "type": "string" }, "type": "array" },
                    "lastInsertRowid": {
                        "format": "int64",
                        "nullable": true,
                        "type": "integer",
                    },
                    "rows": {
                        "items": { "items": {}, "type": "array" },
                        "type": "array",
                    },
                    "rowsAffected": { "format": "int64", "type": "integer" },
                },
                "type": "object",
            },
            "StatementObject": {
                "properties": {
                    "args": {
                        "oneOf": [{ "items": {}, "type": "array" }, {
                            "additionalProperties": {},
                            "type": "object",
                        }],
                    },
                    "sql": { "type": "string" },
                },
                "required": ["sql"],
                "type": "object",
            },
        },
    },
    "info": {
        "description": "API for Smallweb blob storage and SQLite operations",
        "title": "Smallweb API",
        "version": "1.0.0",
    },
    "openapi": "3.0.3",
    "paths": {
        "/blob/{key}": {
            "delete": {
                "operationId": "DeleteBlob",
                "parameters": [{
                    "description":
                        "The blob key. Suffix with / or colon to clear all with prefix.",
                    "in": "path",
                    "name": "key",
                    "required": true,
                    "schema": { "type": "string" },
                }],
                "responses": {
                    "200": {
                        "content": { "text/plain": { "schema": { "type": "string" } } },
                        "description": "Success",
                    },
                    "500": { "description": "Internal Server Error" },
                },
                "summary": "Remove blob item or clear with prefix",
            },
            "get": {
                "operationId": "GetBlob",
                "parameters": [{
                    "description": "The blob key. Suffix with / or colon to list keys.",
                    "in": "path",
                    "name": "key",
                    "required": true,
                    "schema": { "type": "string" },
                }, {
                    "in": "header",
                    "name": "Accept",
                    "schema": { "type": "string" },
                }],
                "responses": {
                    "200": {
                        "content": {
                            "application/json": {
                                "schema": { "items": { "type": "string" }, "type": "array" },
                            },
                            "application/octet-stream": {
                                "schema": { "format": "binary", "type": "string" },
                            },
                            "text/plain": { "schema": { "type": "string" } },
                        },
                        "description": "Success",
                    },
                    "404": { "description": "Not Found" },
                    "500": { "description": "Internal Server Error" },
                },
                "summary": "Get or list blob items",
            },
            "head": {
                "operationId": "HasBlob",
                "parameters": [{
                    "in": "path",
                    "name": "key",
                    "required": true,
                    "schema": { "type": "string" },
                }],
                "responses": {
                    "200": { "description": "Item exists" },
                    "404": { "description": "Item not found" },
                },
                "summary": "Check if blob item exists",
            },
            "put": {
                "operationId": "SetBlob",
                "parameters": [{
                    "in": "path",
                    "name": "key",
                    "required": true,
                    "schema": { "type": "string" },
                }],
                "requestBody": {
                    "content": {
                        "application/octet-stream": {
                            "schema": { "format": "binary", "type": "string" },
                        },
                    },
                    "required": true,
                },
                "responses": {
                    "200": {
                        "content": { "text/plain": { "schema": { "type": "string" } } },
                        "description": "Success",
                    },
                    "500": { "description": "Internal Server Error" },
                },
                "summary": "Set blob item",
            },
        },
        "/sqlite/batch": {
            "post": {
                "operationId": "BatchSQLite",
                "requestBody": {
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/BatchRequest" },
                        },
                    },
                    "required": true,
                },
                "responses": {
                    "200": {
                        "content": {
                            "application/json": {
                                "schema": {
                                    "items": { "$ref": "#/components/schemas/ExecuteResult" },
                                    "type": "array",
                                },
                            },
                        },
                        "description": "Success",
                    },
                    "400": { "description": "Bad Request" },
                    "500": { "description": "Internal Server Error" },
                },
                "summary": "Execute a batch of SQLite statements in a transaction",
            },
        },
        "/sqlite/execute": {
            "post": {
                "operationId": "ExecuteSQLite",
                "requestBody": {
                    "content": {
                        "application/json": {
                            "schema": { "$ref": "#/components/schemas/ExecuteRequest" },
                        },
                    },
                    "required": true,
                },
                "responses": {
                    "200": {
                        "content": {
                            "application/json": {
                                "schema": { "$ref": "#/components/schemas/ExecuteResult" },
                            },
                        },
                        "description": "Success",
                    },
                    "400": { "description": "Bad Request" },
                    "500": { "description": "Internal Server Error" },
                },
                "summary": "Execute a single SQLite statement",
            },
        },
    },
    "servers": [{ "description": "API v1", "url": "/v1" }],
} as const
