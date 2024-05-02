# Coding guidelines

This page collects common comments made during reviews. This is a laundry list of common mistakes, not a comprehensive style guide.

## Previous work

Sticking to the following three guidelines will greatly increase your reviewers’ happiness levels:

- https://go.dev/doc/effective_go
- https://tip.golang.org/doc/comment
- https://github.com/golang/go/wiki/CodeReviewComments

In addition to that, we compiled a list of a few more things that are quite common.


## TL;DR

- Be consistent, concise and check your own work.
- “TODO” and “NOTE” comments should have an issue ID.
- Spelling and grammars matters.
- PRs must be squashed before merging and should have a short and long description separated by an empty newline.
- Use correct variable and import alias names.
- Wrap errors with fmt.Errorf(“...%w”, err) and compare them via errors.Is.
- Contexts must be passed down the call stack and should be the first argument.
- Do not use context.Background() or context.TODO(), except in tests.
- Check Go Docs and comments for validity even in unchanged but affected places.
- All public functions must be tested
- Tests must run with enabled race detection.

## General non-code stuff

- Consistency, consistency, consistency. This applies almost everywhere.
- Review your code yourself before you ask someone else to do it.
- Check what you are about to commit.
- The reviewer opens a comment, and it's the reviewer resolving the comment. You can answer if you have questions or disagree or want to discuss things.
- Avoid changes not related to your ticket. Only do those flybys if they are really absolutely nothing a reviewer could complain about. Typos are usually ok, but everyone likes a clear focus instead of discussing stuff that's actually off-topic.
- Don’t force-push your git branch after someone started reviewing it. Changing the history makes it harder for a reviewer to find the exact diff, e.g. since the reviewer had a look the last time.
- Pull requests that are not meant to be reviewed or are a work in progress SHOULD be set to “draft”.
- A pull request’s single commits MUST be squashed via GitHub when merging. Also check the guidelines for commit messages below.

## General code stuff

- Consistency, it also applies here.
- When leaving TODO comments in the code, make sure you add the related issue to it, e.g.
    ```go
    func foo() {
        // TODO(#999, this should be done later)
    }
    ```
- Longer comments that are more than just normal code comments SHOULD be marked as a NOTE, again with the issue:
    ```go
    // NOTE(#1000): Lorem ipsum dolor sit amet, consetetur
    // sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore
    // magna aliquyam erat, sed diam voluptua. 
    // At vero eos et accusam et justo duo dolores et ea rebum.
    // Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet.
    ```
- Keep interfaces small. This applies to Golang interfaces themselves, but also to a package's and type's exported functions. Same for variables and constants.

- If in doubt, use a pointer (see go-code-reviews).

- Infrastructure types attributes, unless necessary, should not be pointers (e.g. use `string` instead of `*string`) 

- Use constructors to initialize structs if they are more convenient, e.gi.e. less fields required, sets defaults, etc.

- Group struct fields by public/private and purpose, e.g. a mutex stays with the objects it protects.

- If you add a field to a struct with alphabetically sorted fields, keep the fields sorted.

## Commit messages

> (OpenStack has compiled some best-practices. For inspiration: https://wiki.openstack.org/wiki/GitCommitMessages)

The following requirements are only applicable to your final squashed commit. It does not apply to your single commits in your development branch.

- The commit message proposed by GitHub just contains all single commits’ messages and is often too verbose, outdated and not helpful.
- Your commit message SHOULD adhere to the following structure:
    - Provide a brief description of the change in the first line. This does not include the ticket ID.
    - Insert a single blank line after the first line.
    - Provide a detailed description of the change in the following lines, breaking paragraphs where needed.
    - The first line is limited to 50 characters and does not end with a period.
    - Subsequent lines are wrapped at 72 characters.
    - Your commit references the feature/bug it implements/fixes, e.g. by adding a last paragraph with “Implements: <issue ID>” or “Fixes: <issue ID>”. There can be multiple of those ticket references in that paragraph. Other keywords include “Relates-To”.

## Naming conventions

- Use camel-case with capital letters for acronyms, e.g. HTTP, API, etc.
- Import names and aliases SHOULD be precise. v1 isn't. E.g. in the context of K8s, we use corev1, metav1, appsv1 etc.
- Be consistent here, too: Match parameters and type names at package level, e.g. single name for a client parameter across different functions. Match import aliases across files and preferably across the whole project. The importas linter can be used to enforce this.
- Use meaningful names. This is not C. But also not Java. Getters should not be prefixed with Get, see effective_go.
- Prefer “node pool” over “nodepool”; "Data center" is a special case: Variables should have it as one word `datacenter` but comments should have it as two words `// data center`.
- Package names SHOULD be in singular not plural form. E.g. middleware instead of middlewares. From a package consumer point of view it can also be better to have distinct subpackages in middleware instead of having a collection of middlewares that have nothing in common (besides being a middleware). However, no need to change plural names created by code generation tools like kubebuilder that don’t provide configuration options. We don’t want to fight our tools.

## Spelling & grammar

- Aim to write good English. There are tools online that can help you with this.
- Inside comments, write things the way they're spelled/stylized. Ionos Cloud -> IONOS Cloud; CoreDns -> CoreDNS; Ip -> IP. A special case is "identifier": here we always use ID.


## Errors

- You SHOULD wrap errors if there is additional useful information.
- Error variables SHOULD begin with Err and error types SHOULD end with Error.
- Compare errors for equality with errors.Is. (which is different from errors.As )

## Godoc

- When you have godocs for an interface methods AND its implementation(s), changing the godocs in one place might require changes to the other place(s), too.
- Exported stuff needs godocs unless they implement a built-in interface, e.g. error or are getters or setters.
Context
- Functions that accept a context MUST pass it down the call stack instead of using context.Background() or context.TODO(), unless there is a good reason to do so which MUST then be added as comment.
- If a context is not available, change the parent function signature to accept one and pass it on. Do not use context.Background() or context.TODO().

## Refactoring

- When you rename a function, you might also need to rename corresponding tests.
- Use your editor "Search and Replace" functionality. Finding all the places manually has proven to be error-prone. Check that you didn't replace more than intended.
- When moving stuff around, make sure that code comments stay valid and close to the code they refer to.

## Testing

- All public functions of a package SHOULD be tested even if they are trivial.
- If you add or change logic, you probably also need to touch the unit tests.
- No need to retest stuff in integration tests that were already covered by unit tests.
- Code coverage is good, but it is not required to get it to 100%. Cover the important code paths. Simple if err != nil { return fmt.Errorf("...: %w", err) }  occurrences don’t need to be covered, we have the nilerr linter for that.
- Tests MUST run with enabled race detection.
- Your table-driven test cases SHOULD NOT share any lines, so please avoid }, {
- Don’t overcomplicate table-driven tests. If the setup becomes increasingly complex consider refactoring instead of extending the test. Each control flow statement in the test body increases the probability that a separate test function should be implemented.
- If you need a context in tests, use context.Background().
- When mocking calls, try to be precise. mock.Anything is fine for complicated data structures we don't care too much about (e.g. contexts), but don't overuse it.
- github.com/stretchr/testify has lots of convenient functions. Use them. There’s more than Equal(). Same goes for gomega/ginkgo. An IDE might help you find those helpers. 
- Use testify’s require instead of assert if all assertions afterwards would fail anyway and don't provide more insights for debugging the test. Test setup code (like writing files or initializing data) MUST use require instead of ignoring errors.
- Tests that do repetitive setup SHOULD use the suite package.
- If your test suite only has a single test, then there might be no reason to use a suite at all.
- Tests SHOULD make use of t.Cleanup() for tear down steps. It is not widely used yet, but we should gradually migrate.
- Tests SHOULD use t.Parallel() ONLY when the tested code does not depend on unlocked global state like function variables or singletons. Always check if imported code depends on global state and is safe to be used concurrently. testify’s suites have problems with running tests in parallel: https://github.com/stretchr/testify/issues/187
- Helper test functions SHOULD use t.Helper()
- Create a variable at the appropriate scope if you find yourself initializing one over and over.
- It’s okay to drop the error for “mock” functions, e.g. wrapping io.Writer.Write. Just assign it to the _ variable.

## Debugging

- Rather use the debugger than instrumenting your code with fmt.Println calls.

## CRDs, controllers, operators

- Optional fields in CRDs SHOULD also use omitempty in their json struct tags.
- When writing kubebuilder RBAC markers think about whether you use a caching client. If so, each GET will also issue a LIST and a WATCH.


## Logging

- Use lowerCamelCase keys for log fields, e.g: requestID, durationMS.
- Logging keys set by WithValues should not persist past their intended use/scope, e.g. continuing to use a request body.
- Logging errors SHOULD happen with a format that will show a potential corresponding stack trace, e.g. %+v.
- Use the proper log level:
    - Trace: Information provided as this level is only relevant for the developer. There is no rule for a proper amount of log messages, but it should not replace your debugger during development. It shall help to understand the program flow without a debugger in a production environment. 
    - Debug: Information that is diagnostically helpful to more people than just developers (IT, sysadmins, etc.).
    - Info: Generally useful information to log (service start/stop, configuration assumptions, etc). This is the standard log level and must keep a good signal-to-noise ratio.
    - Warning: Not an error but likely more important than an informational event. This could for example be a deprecation warning. 
    - Error: Any error, regardless if an operation failed, or the service recovered a panic. An error might require manual work to fix, or indicate a transient error, that recovers automatically. But it still indicates non-normal program flow.
