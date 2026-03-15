export async function GET() {
  return Response.json({ service: "admin-web", status: "ok", version: "0.1.0" });
}

