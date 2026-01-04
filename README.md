# AWS MCP Credential Proxy

1. Reads the `AWS_CONTAINER_AUTHORIZATION_TOKEN` and `AWS_CONTAINER_CREDENTIALS_FULL_URI` environment variables passed from running [AWS Vault] in server mode.

1. Exchanges those for traditional STS credentials (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and `AWS_SESSION_TOKEN`).

1. Passes those credentials down the line to the next command that only understands traditional STS credentials.

## Example

```bash
aws-vault exec --duration=15m --ecs-server --region=us-east-2 --lazy {ROLE} -- \
  aws-mcp-credential-proxy -- \
    docker mcp gateway run \
      --servers=aws-api \
      --servers=aws-core-mcp-server \
      --servers=aws-documentation \
      --servers=aws-terraform \
      --tools=call_aws \
      --tools=fetch_agentcore_doc \
      --tools=manage_agentcore_gateway \
      --tools=manage_agentcore_memory \
      --tools=manage_agentcore_runtime \
      --tools=mcp-add \
      --tools=mcp-create-profile \
      --tools=mcp-find \
      --tools=prompt_understanding \
      --tools=recommend \
      --tools=search_agentcore_docs \
      --tools=suggest_aws_commands \
;
```

Helps address the fact that [AWS MCP servers don't (yet) support AWS Container Credentials](https://github.com/awslabs/mcp/issues/2043).

## Refreshes

[AWS Vault]: https://github.com/ByteNess/aws-vault#readme
