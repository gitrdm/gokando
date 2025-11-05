# Frequently Asked Questions

## General

### What is gokando?

[![Version](https://img.shields.io/badge/version-1.0.1-blue.svg)](https://github.com/gitrdm/gokando/releases)

### What are the system requirements?

- Go 1.21 or later
- No external dependencies required

### How do I install gokando?

```bash
go get github.com/gitrdm/gokando@latest
```

### Is gokando production ready?

Yes, gokando is designed for production use with a focus on reliability, performance, and safety.

## Performance

### What are the performance characteristics?

gokando is designed for high performance with minimal overhead. See our [performance documentation](performance.md) for detailed benchmarks.

### How does gokando handle memory allocation?

gokando is designed to minimize allocations in hot paths. Most operations are allocation-free in steady state.

## Usage

### Can I use gokando with other libraries?

Yes, gokando is designed to work well with the standard library and other Go packages.

### Are there any gotchas I should know about?

See our [best practices guide](best-practices.md) for common patterns and pitfalls to avoid.

## Support

### How do I get help?

- Check this FAQ
- Browse the [documentation](../README.md)
- Search [existing issues](https://github.com/gitrdm/gokando/issues)
- Open a [new issue](https://github.com/gitrdm/gokando/issues/new)

### How do I report a bug?

Please open an issue on GitHub with:

- A clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Your Go version and OS

### How do I request a feature?

Open an issue on GitHub with:

- A clear description of the feature
- Why it would be useful
- Proposed API design (if applicable)
