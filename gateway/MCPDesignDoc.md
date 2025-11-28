# MCP Proxy support in API Platform Gateway

# What is the problem we are trying to solve and why should it be solved?

As organizations move towards building more and more AI Agents, enterprise adoption of Model Context Protocol (MCP) has become vital. With the rapid advancement of MCP based development and solutions, organizations are now faced with a whole new challenge of managing the MCP servers they use. These newly created vectors for risk and complexity include,

* Security concerns such as addressing how to manage authentication and authorization for different MCP capabilities.  
* Observability concerns such as ensuring adequate visibility into MCP interactions and auditing.  
* Traffic management concerns such as maintaining consumption limits per tools, resources, and prompts in the MCP server.

While many tools exist to address these individual challenges, managing them in a decentralized way becomes cumbersome in an enterprise setting. The API Platform aims to deliver an enterprise-grade MCP gateway that enables organizations to centrally proxy MCP server traffic and enforce governance over MCP server behavior.

# Who are we solving it for?

* AI Developers who have developed their own MCP servers and want to apply QoS such as security, rate limiting, etc.  
* Administrators who want to give internal teams access to external MCP servers by enforcing security and limits.

# Proposed Solution

The proposed solution is to allow deploying MCP proxies in the gateway. The Gateway Controller will expect an artifact similar to the following. It will process and translate the provided artifact in order to configure the Router.

* The artifact creation can be done manually. We will also provide a simple command line tool or a script which will generate the config once the MCP server url is provided.  
* MCP capabilities (tools, resources, prompts) will be treated as first class entities in the configuration instead of operations unlike in an API proxy.  
* The relevant operations will be derived internally by the gateway based on the provided MCP specification version and will be used to create the routes.

```
version: api-platform.wso2.com/v1
kind: mcp
data:
  name: my-mcp-server
  version: v1.0
  context: /mymcp
  specVersion: 2025-06-18
  upstream:
    - url: https://mcp-server-backend:5000
  policies:
    - name: mcpAuthentication         
      params:
        key1: value1
        key2: value2
  tools:
    - name: ...
      title: ...
      description: ...
      inputSchema: ...
      policies: ...
  resources:
       ....
  prompts:
       ...
```

## User Flow

![][image1]