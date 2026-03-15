export async function GET() {
  return Response.json({ service: "chat-web", status: "ok", version: "0.1.0" });
}

