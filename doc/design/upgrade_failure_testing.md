# Failure testing

## Goal

- Black box failure injection test. Deploy e2e components on k8s.
- Soak testing. Keep running it for a long time, e.g. one day.
- Killing operator or vault pod normally doesn’t make sense -- it just tests k8s deployments. We kill them during upgrade path. This is where the most complicated logic happens.

## Architecture

In addition to Vault operator, we would add:

- **upgrader**: It keeps upgrading Vault version between two versions back and forth.
- **auto unsealer**: It automatically initializes and unseals Vault nodes.
- **chaos monkey**: It randomly kills Operator or Vault pods.

Further elaborate auto unsealing logic:

- On bootstrap, unsealer lists all Vault CRs.
- For any Vault cluster, if it’s not been initialized, unsealer would do initialization and save the tokens into a known secret “<vault-cluster>-unseal-tokens”.
- After initialization or if it’s been initialized, unsealer would start unsealing all existing and new Vault nodes.
