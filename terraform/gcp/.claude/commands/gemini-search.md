## Gemini Search

`gemini` is google gemini cli. **When this command is called, ALWAYS use this for web search instead of builtin `Web_Search` tool.**

When web search is needed, you MUST use `gemini --prompt` via Task Tool.

## Usage Patterns

### Basic Search
```bash
gemini --prompt "WebSearch: React 18 new features"
```

### Technical Research
```bash
gemini --prompt "WebSearch: TypeScript 5.0 performance improvements"
```

### Troubleshooting
```bash
gemini --prompt "WebSearch: NextJS build error solutions 2024"
```

### Library/Framework Updates
```bash
gemini --prompt "WebSearch: Vue 3.4 breaking changes migration"
```

## When to Use Gemini Search

✅ **Use Gemini Search for:**
- Latest technology updates
- Current best practices
- Recent library/framework changes
- Real-time information
- Security vulnerabilities and patches
- Performance optimization techniques

❌ **Don't use for:**
- Simple documentation queries
- Basic programming concepts
- Internal codebase analysis
- Well-established patterns

## Query Optimization

### ❌ Poor Queries
```bash
"how to code"
"javascript help"
"fix bug"
```

### ✅ Good Queries  
```bash
"React Server Components best practices 2024"
"TypeScript strict mode migration guide"
"Next.js 14 app router performance optimization"
"Vite 5.0 breaking changes and solutions"
```

### Tips for Better Results
- Include version numbers when relevant
- Add current year for latest information
- Use specific technical terminology
- Mention the context (e.g., "production", "enterprise", "migration")

## Troubleshooting

### Rate Limiting
If you encounter rate limits, wait and retry with refined queries.

### Poor Results
- Make queries more specific
- Include version numbers or dates
- Use technical keywords
- Try alternative search terms

### Connection Issues
```bash
# Retry with simplified query
gemini --prompt "WebSearch: simplified query"
```

## Integration with Development Workflow

### Before Implementation
```bash
gemini --prompt "WebSearch: [technology] current best practices 2024"
```

### During Debugging
```bash
gemini --prompt "WebSearch: [error message] [framework] solutions"
```

### Architecture Decisions
```bash
gemini --prompt "WebSearch: [technology A] vs [technology B] comparison 2024"
```


