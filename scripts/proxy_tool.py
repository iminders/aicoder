import json

import requests
from flask import Flask, Response, request

app = Flask(__name__)

# 目标：你本地运行 LLM 的真实端口
TARGET_SERVER = "http://127.0.0.1:10002"

@app.route('/v1/chat/completions', methods=['POST'])
@app.route('/chat/completions', methods=['POST'])
def proxy_deepseek():
    req_data = request.get_json(silent=True)

    print("=" * 80 + f" number of messages: {len(req_data['messages'])} " + "=" * 80 + "\n")
    print(json.dumps(req_data, indent=4, ensure_ascii=False))

    # --- 打印逻辑保持不变 ---
    print(f"\n🚀 [Intercepted] Model: {req_data.get('model')}")

    # 1. 清洗 Headers：移除会导致冲突的传输层 Header
    excluded_headers = ['content-encoding', 'content-length', 'transfer-encoding', 'connection', 'host']
    headers = {
        k: v for k, v in request.headers
        if k.lower() not in excluded_headers
    }

    # 2. 转发请求
    is_stream = req_data.get('stream', False) if req_data else False

    try:
        resp = requests.post(
            f"{TARGET_SERVER}{request.full_path}",
            json=req_data,
            headers=headers,
            stream=is_stream,
            timeout=3000  # local LLM 推理较慢，建议增加超时
        )

        # 3. 构造响应并清洗返回的 Headers
        def generate():
            for chunk in resp.iter_content(chunk_size=None):  # chunk_size=None 保持原始分块
                yield chunk

        # 同样移除返回时的冲突 Header
        resp_headers = [
            (k, v) for k, v in resp.raw.headers.items()
            if k.lower() not in excluded_headers
        ]

        return Response(
            generate() if is_stream else resp.content,
            status=resp.status_code,
            headers=resp_headers
        )
    except Exception as e:
        print(f"❌ Proxy Error: {e}")
        return Response(json.dumps({"error": str(e)}), status=500, content_type='application/json')


if __name__ == '__main__':
    # 启动代理，监听 10003
    print("Proxy is running on http://127.0.0.1:10003")
    print(f"Forwarding to Local LLM at {TARGET_SERVER}")
    app.run(port=10003, debug=False)
