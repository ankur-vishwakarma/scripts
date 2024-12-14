import asyncio
from anthropic.types.beta import BetaTextBlockParam
from loop import sampling_loop, APIProvider
import os
from flask import Flask, request, jsonify
from functools import wraps

app = Flask(__name__)

def async_route(f):
    @wraps(f)
    def wrapped(*args, **kwargs):
        return asyncio.run(f(*args, **kwargs))
    return wrapped

async def send_prompt(prompt: str):
    messages = []

    messages.append({
        "role": "user",
        "content": [
            BetaTextBlockParam(type="text", text=prompt)
        ]
    })

    print("prompt", prompt)
    messages = await sampling_loop(
            model="claude-3-5-sonnet-20241022",
            provider=APIProvider.ANTHROPIC,
            system_prompt_suffix="",
            messages=messages,
            output_callback=lambda x: print(f"Output: {x}"),
            tool_output_callback=lambda x, y: print(f"Tool output: {x}"),
            api_response_callback=lambda x, y, z: print(f"API response: {y}"),
            api_key=os.getenv("ANTHROPIC_API_KEY", ""),
            only_n_most_recent_images=3
        )

    return messages

@app.route("/prompt", methods=["POST"])
@async_route
async def handle_prompt():
    data = request.get_json()

    if not data or 'prompt' not in data:
        return jsonify({'error': 'No prompt provided'}), 400

    try:
        prompt = data['prompt']

        messages = await send_prompt(prompt)
        return jsonify({'status': 'success', 'messages': messages})

    except Exception as e:
        return jsonify({'error': str(e)}), 500

if __name__ == "__main__":
    app.run(host='0.0.0.0', port=7474, debug=True)
