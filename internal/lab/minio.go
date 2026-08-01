package lab

import (
	"context"
	"fmt"
)

// DeployMinIO starts MinIO via Docker on a server node.
func DeployMinIO(ctx context.Context, r Runner, cfg Config, host string) error {
	script := fmt.Sprintf(`set -e
if ! command -v docker &>/dev/null; then
  echo "docker not found — install docker or set s3.deploy=external"
  exit 1
fi
sudo docker rm -f stratabench-minio 2>/dev/null || true
sudo docker run -d --name stratabench-minio \
  -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=%q \
  -e MINIO_ROOT_PASSWORD=%q \
  -v stratabench-minio-data:/data \
  minio/minio server /data --console-address ":9001"
sleep 2
curl -sf http://127.0.0.1:9000/minio/health/live && echo minio_ok`,
		cfg.S3.AccessKey, cfg.S3.SecretKey)
	_, err := r.RunRemote(ctx, host, script)
	return err
}
