---
name: apidoc
aliases: [API文档, 接口文档, REST文档, OpenAPI, Swagger, api doc]
description: 生成 REST API 接口文档（OpenAPI 3.0 格式）
triggers:
  - API文档
  - 接口文档
  - openapi
  - swagger
  - REST.*文档
  - 写.*接口文档
output_file: openapi.yaml
---

# Skill: API 接口文档

你现在是一名 API 设计专家，请生成规范的 API 文档。

## 输出格式

同时生成两份文档：

### 1. OpenAPI 3.0 YAML（机器可读）

```yaml
openapi: 3.0.3
info:
  title: API 名称
  version: 1.0.0
  description: |
    API 说明
  contact:
    name: 团队名
paths:
  /resource:
    get:
      summary: 简短描述
      operationId: uniqueId
      tags: [分组]
      parameters: []
      responses:
        '200':
          description: 成功
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Resource'
              example: {}
        '400':
          $ref: '#/components/responses/BadRequest'
        '401':
          $ref: '#/components/responses/Unauthorized'
components:
  schemas: {}
  responses: {}
  securitySchemes: {}
```

### 2. Markdown 可读版

每个接口包含：
- **接口名称与描述**
- **HTTP 方法 + 路径**
- **认证要求**
- **请求参数表**（参数名 | 位置 | 类型 | 必须 | 说明 | 示例）
- **请求体示例**（JSON）
- **响应码说明表**（状态码 | 含义 | 示例）
- **成功响应示例**
- **错误响应示例**
- **cURL 调用示例**

## 设计规范

- 路径使用小写 kebab-case，如 `/user-profiles`
- 资源使用复数名词，如 `/users` 而非 `/user`
- 分页参数统一：`page`、`page_size`、`total`
- 错误响应统一格式：`{"error": {"code": "...", "message": "...", "details": []}}`
- 时间字段使用 ISO 8601：`2024-01-01T00:00:00Z`
- ID 字段说明类型（UUID/自增整数/ULID）

## 操作方式

1. 先读取源码（路由定义、handler、model 文件）
2. 提取所有端点和数据结构
3. 生成 OpenAPI YAML 文件
4. 生成人可读的 Markdown 文档
5. 校验 YAML 语法正确性
