# Contributing to Titan Infrastructure OS

First off, thank you for considering contributing to Titan Infrastructure OS! It's people like you that make Titan OS such a great platform.

## Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the issue list as you might find out that you don't need to create one. When you are creating a bug report, please include as many details as possible:

- **Use a clear and descriptive title**
- **Describe the exact steps which reproduce the problem**
- **Provide specific examples to demonstrate the steps**
- **Describe the behavior you observed after following the steps**
- **Explain which behavior you expected to see instead and why**
- **Include screenshots and animated GIFs if possible**
- **Include your environment details** (OS, Docker version, Node version, etc.)

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, please include:

- **Use a clear and descriptive title**
- **Provide a step-by-step description of the suggested enhancement**
- **Provide specific examples to demonstrate the steps**
- **Describe the current behavior and the expected behavior**
- **Explain why this enhancement would be useful**

### Pull Requests

- Fill in the required template
- Follow the TypeScript/Go styleguides (see below)
- Include appropriate test cases
- Update documentation as needed
- End all files with a newline

## Development Setup

1. **Fork the repository** on GitHub

2. **Clone your fork locally**:
   ```bash
   git clone https://github.com/YOUR_USERNAME/TitanOS.git
   cd TitanOS
   ```

3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/ChaitanyaJoshi1769/TitanOS.git
   ```

4. **Install dependencies**:
   ```bash
   make install
   ```

5. **Start development stack**:
   ```bash
   make dev
   ```

6. **Create a branch** for your work:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Workflow

### Before You Start

1. Check the [Phase Roadmap](docs/ARCHITECTURE.md) to understand current priorities
2. Look for existing issues or discussions related to your work
3. Comment on issues to let maintainers know you're working on something

### Code Style

#### TypeScript

- Use **2-space indentation**
- Use **double quotes** for strings
- Use **trailing commas** in multi-line objects/arrays
- Use **PascalCase** for classes and types
- Use **camelCase** for functions and variables
- Use **UPPER_SNAKE_CASE** for constants

```typescript
// Good
interface UserProfile {
  firstName: string;
  lastName: string;
  email: string,
}

function getUserProfile(userId: string): UserProfile {
  // implementation
}

const MAX_RETRIES = 3;
```

#### Go

- Follow [Effective Go](https://golang.org/doc/effective_go) guidelines
- Use **gofmt** for formatting
- Use **PascalCase** for exported identifiers
- Use **camelCase** for unexported identifiers
- Document all exported types and functions

```go
// Good
func ScheduleTask(ctx context.Context, task *Task) error {
  // implementation
}

type TaskScheduler struct {
  // fields
}

const maxRetries = 3
```

### Running Tests

```bash
# Run all tests
make test

# Run only unit tests
make test-unit

# Run only integration tests
make test-integration

# Run with coverage
npm run test:coverage
```

### Code Quality

```bash
# Run linters
make lint

# Format code
make format

# Type check (TypeScript)
make type-check
```

### Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Types:
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only
- `style`: Changes that don't affect code meaning (formatting, missing semicolons, etc.)
- `refactor`: Code change that neither fixes a bug nor adds a feature
- `perf`: Code change that improves performance
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to build process, dependencies, tooling, etc.

Examples:
```
feat(scheduler): add resource affinity rules
fix(gateway): handle authentication timeout correctly
docs(api): update GraphQL schema documentation
test(workflow): add integration test for retry logic
```

## Testing Guidelines

- **Unit Tests**: Test individual functions and components in isolation
- **Integration Tests**: Test components working together with real dependencies
- **E2E Tests**: Test complete workflows from user perspective
- **Load Tests**: Benchmark performance and scalability
- **Chaos Tests**: Verify recovery and resilience

### Test File Organization

```
services/scheduler/
├── scheduler.go
├── scheduler_test.go        # Unit tests
├── integration_test.go      # Integration tests (requires Docker)
└── benchmark_test.go        # Benchmarks
```

### Writing Tests

```typescript
// Jest example
describe("Scheduler", () => {
  it("should schedule tasks in FIFO order", () => {
    const scheduler = new Scheduler();
    scheduler.schedule(task1);
    scheduler.schedule(task2);
    
    expect(scheduler.next()).toBe(task1);
    expect(scheduler.next()).toBe(task2);
  });
});
```

```go
// Go testing example
func TestScheduleTask(t *testing.T) {
  scheduler := NewScheduler()
  task := &Task{ID: "test-1"}
  
  err := scheduler.Schedule(context.Background(), task)
  if err != nil {
    t.Fatalf("Schedule failed: %v", err)
  }
  
  retrieved := scheduler.Next()
  if retrieved.ID != task.ID {
    t.Errorf("Expected task ID %s, got %s", task.ID, retrieved.ID)
  }
}
```

## Documentation

- Update [README.md](README.md) if you change functionality
- Update relevant [docs/](docs/) files
- Add code comments for non-obvious logic
- Update API documentation if you change APIs
- Include examples for new features

## Submitting Changes

1. **Ensure all tests pass**:
   ```bash
   make test
   make lint
   make type-check
   ```

2. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```

3. **Create a Pull Request** on GitHub:
   - Reference any related issues
   - Describe what changes you made and why
   - Mention any breaking changes

4. **Address review feedback**:
   - Make requested changes
   - Commit with message like `chore: address review feedback`
   - No need to force push if making small changes

## Review Process

- Maintainers will review your PR within 2-3 business days
- We require at least one approval before merging
- All CI checks must pass
- Code coverage should not decrease

## Additional Notes

### Issue and Pull Request Labels

- `bug`: Something isn't working
- `enhancement`: New feature or request
- `documentation`: Improvements or additions to documentation
- `good first issue`: Good for newcomers
- `help wanted`: Extra attention is needed
- `question`: Further information is requested
- `wontfix`: This will not be worked on

### Phases and Priorities

See [ROADMAP.md](ROADMAP.md) for current phase. We prioritize:
1. **Phase 0**: Foundation (current focus)
2. **Phases 1-3**: Core infrastructure
3. **Phases 4-6**: Enterprise features
4. Etc.

### Where Can I Get Help?

- **Documentation**: [docs/](docs/) folder
- **Issues**: Check existing [GitHub Issues](https://github.com/ChaitanyaJoshi1769/TitanOS/issues)
- **Discussions**: Ask in [GitHub Discussions](https://github.com/ChaitanyaJoshi1769/TitanOS/discussions)

## Recognition

Contributors will be recognized in:
- [CONTRIBUTORS.md](CONTRIBUTORS.md) file
- GitHub contributors page
- Release notes for major contributions

Thank you for contributing to Titan Infrastructure OS! 🚀
