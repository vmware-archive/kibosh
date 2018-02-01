package config_test

const valid_vcap_services = `
{
  "user-provided": [
    {
      "credentials": {
        "kubeconfig": {
          "apiVersion": "v1",
          "clusters": [
            {
              "cluster": {
                "certificate-authority-data": "bXktZmFrZWNlcnQ=",
                "server": "https://example.com:33071"
              },
              "name": "kubo-cluster"
            }
          ],
          "contexts": [
            {
              "context": {
                "cluster": "kubo-cluster",
                "user": "7dd1424e-c44c-4090-ae0d-3c92a8abe52d"
              },
              "name": "kubo-cluster"
            }
          ],
          "current-context": "kubo-cluster",
          "kind": "Config",
          "preferences": {
          },
          "users": [
            {
              "name": "7dd1424e-c44c-4090-ae0d-3c92a8abe52d",
              "user": {
                "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.TJVA95OrM7E2cBab30RMHrHDcEfxjoYZgeFONFh7HgQ"
              }
            }
          ]
        }
      },
      "label": "kubo-odb",
      "plan": "demo",
      "name": "my-kubernetes",
      "tags": ["pivotal", "kubernetes", "k8s"]
    }
  ]
}
`
