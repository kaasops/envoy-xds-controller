import json
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path


PATCH = Path(__file__).resolve().parents[2] / "docs/examples/argocd-health/argocd-cm.patch.yaml"


class VirtualServiceHealth(unittest.TestCase):
    def test_publication_outcomes(self):
        cases = [
            ("new", None, "Progressing"),
            ("legacy", {"invalid": False}, "Progressing"),
            ("legacy-invalid", {"invalid": True, "message": "old error"}, "Progressing"),
            ("stale", {"observedGeneration": 1, "invalid": True}, "Progressing"),
            ("missing-result", {"observedGeneration": 2}, "Progressing"),
            ("invalid", {"observedGeneration": 2, "invalid": True, "message": "duplicate listener address"}, "Degraded"),
            ("published-with-warning", {"observedGeneration": 2, "invalid": False, "message": "SDS secret skipped"}, "Healthy"),
            ("recovered", {"observedGeneration": 2, "invalid": False}, "Healthy"),
        ]
        with tempfile.TemporaryDirectory() as tmp:
            directory = Path(tmp)
            shutil.copyfile(PATCH, directory / "health.patch.yaml")
            base = {
                "apiVersion": "v1", "kind": "ConfigMap",
                "metadata": {"name": "argocd-cm", "namespace": "argocd"},
                "data": {"existing.setting": "preserve-me"},
            }
            (directory / "base.yaml").write_text(json.dumps(base))
            (directory / "kustomization.yaml").write_text(
                "resources:\n- base.yaml\npatches:\n- path: health.patch.yaml\n"
                "  target: {version: v1, kind: ConfigMap, name: argocd-cm, namespace: argocd}\n"
            )
            config = subprocess.check_output(["kubectl", "kustomize", tmp], text=True)
            self.assertIn("existing.setting: preserve-me", config)
            cm = directory / "argocd-cm.yaml"
            cm.write_text(config)

            for name, status, expected in cases:
                with self.subTest(name=name):
                    resource = {
                        "apiVersion": "envoy.kaasops.io/v1alpha1", "kind": "VirtualService",
                        "metadata": {"name": "demo", "generation": 2},
                    }
                    if status is not None:
                        resource["status"] = status
                    manifest = directory / "vs.yaml"
                    manifest.write_text(json.dumps(resource))
                    result = subprocess.check_output([
                        "argocd", "admin", "settings", "resource-overrides", "health",
                        str(manifest), "--argocd-cm-path", str(cm), "--loglevel", "error",
                    ], text=True).splitlines()
                    self.assertIn("STATUS: " + expected, result)
                    if expected != "Progressing" and status.get("message"):
                        self.assertIn("MESSAGE: " + status["message"], result)


if __name__ == "__main__":
    unittest.main()
