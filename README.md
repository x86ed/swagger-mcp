# swagger-mcp

## Overview
`swagger-mcp` is a tool designed to scrape Swagger UI by extracting the `swagger.json` file and dynamically generating well-defined mcp tools at runtime. These tools can be utilized by the MCP client for further tool selection.

## 📽️ Demo Video  
Check out demo video showcasing the project in action:  
[![Watch the Demo](https://img.shields.io/badge/LinkedIn-Demo-blue?style=for-the-badge&logo=linkedin)](https://www.linkedin.com/posts/danish-j-sheikh_mcp-modelcontextprotocol-llm-activity-7300786040389218304-qfNk?utm_source=share&utm_medium=member_ios&rcm=ACoAAEGFv8IB3uEbMighmc1gppVW4RcC1OUoSC4)  

## 🙌 Support  
If you find this project valuable, please support me on **LinkedIn** by:  
- 👍 Liking and sharing our demo post  
- 💬 Leaving your thoughts and feedback in the comments  
- 🔗 Connecting with us for future updates  

Your support on LinkedIn will help me reach more people and improve the project!  

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

## 🛠️ Need Help  
I am working on **improving tool definitions** to enhance:  
✅ **Better error handling** for more accurate responses  
✅ **LLM behavior control** to ensure it relies **only on API responses** and does not use its own memory  
✅ **Preventing hallucinations** and **random data generation** by enforcing strict data retrieval from APIs  

If you have insights or suggestions on improving these aspects, please contribute by:  
- **Sharing your experience** with similar implementations  
- **Suggesting modifications** to tool definitions  
- **Providing feedback** on current limitations  

Your input will be invaluable in making this tool more reliable and effective! 🚀  
