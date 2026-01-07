# Community Fork: Actively Maintained Webhook Server

## Introduction

Hello webhook community! 👋

I wanted to share information about an actively maintained fork of this project that may be of interest to users looking for:
- Regular security updates and vulnerability fixes
- Active development and maintenance
- Enhanced features and improvements
- Comprehensive documentation (English & Chinese)
- Modern toolchain and dependency updates

## About the Fork

**Repository**: [soulteary/webhook](https://github.com/soulteary/webhook)

This is a community-maintained fork that started from version 2.8.0 and has been actively developed to version **4.9.0**. The fork focuses on:

- 🔒 **Security**: Regular security updates, vulnerability fixes, and enhanced security features (command path whitelisting, argument validation, secure logging)
- 🔧 **Maintenance**: Active development, dependency updates, and bug fixes
- ✨ **Features**: Community-driven improvements including Prometheus metrics, rate limiting, i18n support, and more
- 📚 **Documentation**: Comprehensive bilingual documentation (English and Chinese)

## Key Improvements

### Architecture & Code Quality
- **Modular Architecture**: Refactored from single-file design to modular structure with clear separation of concerns
- **Go Version**: Upgraded from Go 1.14 to Go 1.25
- **Dependencies**: All dependencies updated to latest stable versions
- **Code Quality**: Improved error handling, concurrency safety, and test coverage

### New Features
- 🌍 **Internationalization**: Full bilingual support (English & Chinese)
- 📊 **Prometheus Metrics**: Built-in metrics endpoint (`/metrics`) and health check (`/health`)
- ⚡ **Rate Limiting**: Configurable rate limiting middleware
- ✅ **Configuration Validation**: Built-in config validation command
- 🔍 **Request ID Tracking**: Full request lifecycle tracking for better debugging
- 🛡️ **Security Enhancements**: Command path whitelisting, argument validation, strict mode
- 📝 **Structured Logging**: Improved logging system with log levels and file support
- 🔄 **Graceful Shutdown**: Proper signal handling and graceful shutdown support

### Documentation
- Complete API reference
- Security best practices guide
- Performance tuning guide
- Migration guide
- Troubleshooting guide
- Practical examples and use cases
- Bilingual documentation (English & Chinese)

## Compatibility

The fork maintains **backward compatibility** with existing webhook configurations. Most configurations from the original project should work without modification.

## Migration

If you're interested in trying the fork:

1. **Backup your configuration** files
2. **Test in a non-production environment** first
3. **Review the migration guide**: [Migration Guide](https://github.com/soulteary/webhook/blob/main/docs/en-US/Migration-Guide.md)
4. **Check the refactoring report** for detailed changes: [Refactoring Report](https://github.com/soulteary/webhook/blob/main/docs/en-US/REFACTORING_REPORT.md)

## Installation Options

### Pre-built Binaries
Download from [Releases](https://github.com/soulteary/webhook/releases)

### Docker
```bash
docker pull soulteary/webhook:latest
```

### Build from Source
```bash
git clone https://github.com/soulteary/webhook.git
cd webhook
go build
```

## Resources

- **Repository**: https://github.com/soulteary/webhook
- **Documentation**: 
  - [English](https://github.com/soulteary/webhook/tree/main/docs/en-US)
  - [Chinese](https://github.com/soulteary/webhook/tree/main/docs/zh-CN)
- **Refactoring Report**: [Detailed comparison with original 2.8.0](https://github.com/soulteary/webhook/blob/main/docs/en-US/REFACTORING_REPORT.md)
- **Security Policy**: [Security features and reporting](https://github.com/soulteary/webhook/blob/main/SECURITY.md)

## Statistics

- **Version**: 2.8.0 → 4.9.0
- **Files Changed**: 590 files
- **Lines Added**: +278,389
- **Major Refactoring**: Complete architecture modernization

## Note

This fork is not intended to replace the original project, but rather to provide an alternative for users who need:
- Active maintenance and regular updates
- Enhanced security features
- Modern toolchain support
- Comprehensive documentation

The original project remains valuable, and this fork aims to complement it by providing ongoing maintenance and community-driven improvements.

---

**Disclaimer**: This is a community-maintained fork. Please evaluate it based on your specific needs and requirements. Contributions, feedback, and suggestions are welcome!

