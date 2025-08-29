# 🚀 GitHub Repository Setup Guide for CupBot

## 📋 Prerequisites

Before setting up the repository, ensure you have:
- GitHub account
- Git installed on your Windows machine
- Go 1.21.8+ installed
- Admin access to create repositories (if creating new repo)

## 🏗️ Repository Setup Steps

### 1. Create GitHub Repository

#### Option A: New Repository
```bash
# 1. Create repository on GitHub.com
# - Repository name: cupbot
# - Description: "Advanced Telegram Bot for Windows Computer Management"
# - Choose Public or Private
# - Initialize with README: ❌ No (we have our own)
# - Add .gitignore: ❌ No (we have our own)
# - Add license: ✅ Choose appropriate license (MIT recommended)
```

#### Option B: Fork Existing Repository
```bash
# Fork the repository on GitHub and clone your fork
git clone https://github.com/YOUR_USERNAME/cupbot.git
cd cupbot
```

### 2. Initial Repository Configuration

```bash
# Initialize git repository (if creating new)
cd c:\develop\cupbot
git init

# Add all files
git add .

# Create initial commit
git commit -m "Initial commit: CupBot Telegram bot for Windows management

- Complete Telegram bot implementation with button interface
- File manager with configurable drive access
- Desktop screenshot functionality  
- System event monitoring
- Windows Service integration
- Comprehensive testing suite
- CI/CD pipeline with GitHub Actions"

# Add remote origin (replace with your repository URL)
git remote add origin https://github.com/YOUR_USERNAME/cupbot.git

# Push to GitHub
git branch -M main
git push -u origin main
```

### 3. Configure Repository Settings

#### Branch Protection Rules
1. Go to **Settings** → **Branches**
2. Click **Add rule** for `main` branch:
   - ✅ Require a pull request before merging
   - ✅ Require status checks to pass before merging
   - ✅ Require branches to be up to date before merging
   - ✅ Require conversation resolution before merging
   - ✅ Include administrators

#### Required Status Checks
Add these checks (they'll appear after first CI run):
- `test` - Unit and integration tests
- `build` - Windows build verification
- `security` - Security scanning with Gosec
- `lint` - Code quality checks with golangci-lint

## 🔧 Secrets Configuration

Configure the following secrets in **Settings** → **Secrets and variables** → **Actions**:

### Required Secrets
```yaml
# For Codecov integration (optional but recommended)
CODECOV_TOKEN: "your-codecov-token-here"

# For automated releases (if using)
GITHUB_TOKEN: # Automatically provided by GitHub
```

### Environment Variables (Optional)
```yaml
# For testing with real Telegram bot (use with caution)
BOT_TOKEN: "your-test-bot-token"  # Only for testing branches
ADMIN_USER_IDS: "123456789"       # Test admin ID
```

## 📊 Third-Party Integrations

### 1. Codecov Setup (Code Coverage)
1. Visit [codecov.io](https://codecov.io)
2. Sign in with GitHub
3. Add your repository
4. Copy the token to GitHub Secrets as `CODECOV_TOKEN`

### 2. Security Scanning
- **Gosec**: Automatically configured in CI
- **Dependabot**: Enable in **Settings** → **Security & analysis**
- **CodeQL**: Enable in **Settings** → **Security & analysis**

## 🏷️ Release Management

### Semantic Versioning
Use semantic versioning (v1.0.0, v1.1.0, etc.):

```bash
# Create and push a tag
git tag -a v1.0.0 -m "Release v1.0.0: Initial stable release"
git push origin v1.0.0
```

### Automated Releases
The CI pipeline automatically creates releases when you push tags:

1. **Create Tag**: `git tag -a v1.0.0 -m "Release message"`
2. **Push Tag**: `git push origin v1.0.0`
3. **GitHub Action**: Automatically builds and creates GitHub release
4. **Artifacts**: Windows executable attached to release

## 📁 Repository Structure Best Practices

### Issue Templates
Create `.github/ISSUE_TEMPLATE/`:

```markdown
# Bug Report Template (.github/ISSUE_TEMPLATE/bug_report.md)
---
name: Bug report
about: Create a report to help us improve
title: '[BUG] '
labels: bug
assignees: ''
---

**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Go to '...'
2. Click on '....'
3. See error

**Expected behavior**
A clear description of what you expected to happen.

**Environment:**
- OS: Windows [version]
- Go version: [version]
- CupBot version: [version]

**Additional context**
Add any other context about the problem here.
```

### Pull Request Template
Create `.github/pull_request_template.md`:

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Tests pass locally
- [ ] Added new tests for new functionality
- [ ] Manual testing completed

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No new warnings introduced
```

## 🔄 Development Workflow

### Recommended Git Flow
```bash
# 1. Create feature branch
git checkout -b feature/new-functionality

# 2. Make changes and commit
git add .
git commit -m "feat: add new functionality"

# 3. Push branch
git push origin feature/new-functionality

# 4. Create Pull Request on GitHub
# 5. Wait for CI checks and reviews
# 6. Merge to main after approval
```

### Commit Message Convention
Follow [Conventional Commits](https://www.conventionalcommits.org/):

```bash
feat: add screenshot functionality
fix: resolve file manager path traversal
docs: update README with new features
test: add unit tests for events service
ci: update GitHub Actions to latest versions
```

## 🛡️ Security Considerations

### Sensitive Information
- ❌ **Never commit** bot tokens, API keys, or credentials
- ✅ Use GitHub Secrets for sensitive data
- ✅ Add common secret patterns to `.gitignore`
- ✅ Use placeholder values in example configs

### Repository Access
- 🔒 **Private Repository**: For production use
- 🌍 **Public Repository**: For open source projects
- 👥 **Team Access**: Add collaborators with appropriate permissions

## 📈 Monitoring and Maintenance

### Regular Maintenance Tasks
1. **Weekly**: Review and merge dependabot PRs
2. **Monthly**: Update GitHub Actions to latest versions
3. **Quarterly**: Review and update security policies
4. **As Needed**: Update Go version and dependencies

### CI/CD Monitoring
- Monitor GitHub Actions usage and costs
- Review security scan results
- Check test coverage trends
- Monitor build performance

## 🆘 Troubleshooting

### Common Issues

#### CI Fails on Windows
```yaml
# Solution: Ensure correct path separators
run: .\build.bat  # Not ./build.bat
```

#### Tests Fail Due to Missing Dependencies
```yaml
# Solution: Add to CI workflow
- name: Install CGO dependencies
  run: |
    choco install mingw -y
```

#### Security Scan False Positives
```yaml
# Solution: Add to .golangci.yml exclusions
issues:
  exclude-rules:
    - text: "weak cryptographic primitive"
      linters:
        - gosec
```

#### golangci-lint Version Compatibility Issues
```yaml
# Problem: golangci-lint v2.4.0+ has breaking changes
# Solution 1: Use specific compatible version
- name: golangci-lint
  uses: golangci/golangci-lint-action@v6
  with:
    version: v1.55.2  # Stable version
    
# Solution 2: Simplify .golangci.yml configuration
# Remove deprecated linters and use only essential ones
```

## 📚 Additional Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Go CI/CD Best Practices](https://github.com/mvdan/github-actions-golang)
- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Security Best Practices](https://docs.github.com/en/code-security)

---

## ✅ Quick Setup Checklist

- [ ] Repository created on GitHub
- [ ] Local code pushed to repository  
- [ ] Branch protection rules configured
- [ ] Required status checks enabled
- [ ] Codecov integration setup (optional)
- [ ] Issue and PR templates created
- [ ] Security scanning enabled
- [ ] First CI pipeline run successful
- [ ] Release workflow tested
- [ ] Team access configured
- [ ] Documentation updated

**Next Steps**: Create your first pull request to test the complete workflow!