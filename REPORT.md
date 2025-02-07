# Vet Report

## Summary

|           |                       |
|-----------|-----------------------|
| Critical Vulns  | 1  |
| High Vulns  | 0  |
| Other Vulns  | 0  |
| Unpopular Packages  | 3  |
| Major Version Differences  | 1  |
| Manifests | 1 |
| Total Packages  | 55  |
| Exepmted Packages | 0 |




## Results

| Manifest | Ecosystem | Packages | Need Update |
|----------|-----------|----------|--------------------------|
| /home/lol/core/go.mod | Go | 55 | 4 |

## Policy Violation


> No policy violation found or policy not configured during scan


## Remediation Advice

The table below lists advice for dependency upgrade to mitigate one or more
issues identified during the scan.


> /home/lol/core/go.mod

| Package | Update Version | Impact Score | Issues | Tags   |
|---------|----------------|--------------|--------|--------|
| golang.org/x/crypto@0.28.0 | v0.32.0 | 10 | - | [vulnerability]
| github.com/docker/docker@27.3.1+incompatible | v1.13.1 | 2 | - | [drift]
| github.com/cloudwego/base64x@0.1.4 | v0.1.4 | 1 | - | [low popularity]
| github.com/cloudwego/iasm@0.2.0 | v0.2.0 | 1 | - | [low popularity]
| github.com/containerd/log@0.1.0 | v0.1.0 | 1 | - | [low popularity]




