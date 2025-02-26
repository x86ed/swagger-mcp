# swagger-mcp

## Overview
`swagger-mcp` is a tool designed to scrape Swagger UI by extracting the `swagger.json` file and dynamically generating well-defined mcp tools at runtime. These tools can be utilized by the MCP client for further tool selection.

## Prerequisites
To use `swagger-mcp`, ensure you have the following dependencies:
1. **LLM Model API Key / Local LLM**: Requires access to OpenAI, Claude, or Ollama models.
2. **Any MCP Client**: (Used [mark3labs - mcphost](https://github.com/mark3labs/mcphost))

## Installation and Setup
Follow these steps to install and run `swagger-mcp`:

```sh
go install github.com/danishjsheikh/swagger-mcp@latest
swagger-mcp
```

## MCP Configuration
To integrate with `mcphost`, include the following configuration in `.mcp.json`:
```json
{
    "mcpServers":
    {
        "swagger_loader": {
            "command": "mcp-swagger",
            "args": ["<swagger/doc.json_url>"]
        }
    }
}
```

## Demo Flow
1. Some Backend:
    ```sh
    go install go install github.com/danishjsheikh/go-backend-demo@latest 
    go-backend-demo
    ```

2. Ollama
    ```sh
    ollama run llama3.2
    ```

3. MCP Client
    ```sh
    go install github.com/mark3labs/mcphost@latest
    mcphost -m ollama:llama3.2 --config <.mcp.json_file_path>
    ```

## Flow Diagram
![Flow Diagram](https://raw.githubusercontent.com/danishjsheikh/swagger-mcp/refs/heads/main/swagger_mcp_flow_diagram.png)