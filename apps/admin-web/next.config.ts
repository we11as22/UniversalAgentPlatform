import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  transpilePackages: ["@uap/ui", "@uap/config", "@uap/ts-sdk"]
};

export default nextConfig;
