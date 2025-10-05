# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

We take the security of the Claude Agent SDK for Go seriously. If you discover a security vulnerability, please follow these steps:

### Private Disclosure Process

1. **DO NOT** open a public issue
2. Report vulnerabilities privately through GitHub's Security Advisory feature:
   - Go to https://github.com/conneroisu/claude-agent-sdk-go/security/advisories
   - Click "Report a vulnerability"
   - Fill out the form with details

Alternatively, email security reports to: connerohnesorge@gmail.com

### What to Include

Please include the following in your report:
- Description of the vulnerability
- Steps to reproduce the issue
- Potential impact
- Suggested fix (if any)
- Your contact information for follow-up

### Response Timeline

- **Initial Response**: Within 48 hours of report
- **Vulnerability Assessment**: Within 1 week
- **Patch Development**: Depends on severity (Critical: 1-3 days, High: 1-2 weeks, Medium/Low: 2-4 weeks)
- **Public Disclosure**: After patch is released and users have time to update (typically 7-14 days)

### Security Updates

Security patches will be released as:
- Patch releases (e.g., 0.1.1) for the current stable version
- Backports to previous minor versions if still supported
- Security advisories published through GitHub

### Safe Practices

When using this SDK:
- Always validate and sanitize user inputs before passing to Claude
- Use appropriate permission modes to restrict SDK capabilities
- Never commit API keys or sensitive credentials
- Keep dependencies up to date using Dependabot
- Follow the principle of least privilege

## CVE Assignment

For critical vulnerabilities, we will:
- Request CVE assignment through GitHub
- Coordinate disclosure with GitHub Security Lab
- Publish advisories on security mailing lists

## Bug Bounty

We currently do not have a bug bounty program. Security researchers are credited in release notes and CHANGELOG.md.

## Security Best Practices

### For SDK Users
- Use the latest stable version
- Enable MCP permission callbacks for enhanced security
- Audit hooks that execute custom code
- Verify checksums when installing from releases

### For Contributors
- Follow secure coding guidelines in CONTRIBUTING.md
- Run security scanners (gosec, govulncheck) before submitting PRs
- Never commit secrets or API keys
- Sign commits with GPG keys (optional but recommended)

## Contact

For security-related questions or concerns:
- GitHub Security: https://github.com/conneroisu/claude-agent-sdk-go/security
- Email: connerohnesorge@gmail.com
- Public discussions: https://github.com/conneroisu/claude-agent-sdk-go/discussions
