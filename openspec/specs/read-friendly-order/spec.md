## Purpose

Define the baseline read-friendly-order rules for top-level declarations, constructors, methods, and test-file ordering.

## Requirements

### Requirement: init functions appear before other top-level functions
The system SHALL require `init()` functions to appear before other top-level functions in the same file.

#### Scenario: init follows another top-level function
- **WHEN** a file declares a non-`init` top-level function before `init()`
- **THEN** the analyzer reports that `init()` must be placed before the earlier function

### Requirement: Unexported top-level helpers appear below their first top-level consumer
Within a file, the system SHALL require unexported top-level helper functions, constants, variables, and types to appear below the first top-level symbol that depends on them. Exported symbols, `init()`, cyclic helper relationships, and eager-evaluation constant or variable dependency chains SHALL be exempt from this requirement.

#### Scenario: Helper or constant appears before the top-level symbol that uses it
- **WHEN** an unexported helper or constant is declared above the first top-level symbol that depends on it
- **THEN** the analyzer reports that the helper or constant must be moved below that consuming symbol

#### Scenario: Cyclic helpers remain exempt
- **WHEN** two or more unexported helpers depend on each other cyclically before an exported entry point uses them
- **THEN** the analyzer does not report a top-level ordering violation for that cycle

### Requirement: Constructors appear immediately after their type declaration
The system SHALL require a `New<Type>` constructor to appear immediately after the declaration of the matching type in the same file.

#### Scenario: Constructor is separated from its type by other declarations
- **WHEN** a `New<Type>` function appears later in the file than the declaration it constructs, with other declarations in between
- **THEN** the analyzer reports that the constructor must be placed right after that type declaration

### Requirement: Methods appear below methods that depend on them
The system SHALL require a method to appear below another method on the same receiver type when the later method calls it.

#### Scenario: Helper method appears before a method that calls it
- **WHEN** a method appears earlier in the file than another method on the same receiver type that calls it
- **THEN** the analyzer reports that the called method must be moved below the calling method

### Requirement: TestMain appears first in test files
In `_test.go` files, the system SHALL require `TestMain` to appear before test and benchmark functions.

#### Scenario: TestMain follows a test function
- **WHEN** a `_test.go` file declares `TestMain` after a `Test...` or `Benchmark...` function
- **THEN** the analyzer reports that `TestMain` must be placed first in the test file

### Requirement: Generated files and generated testmain stubs are excluded
The system SHALL skip read-friendly-order enforcement for generated files and for files whose names end with `_testmain.go`.

#### Scenario: Generated file is analyzed
- **WHEN** a file is marked as generated or is a generated `_testmain.go` file
- **THEN** the analyzer does not report read-friendly-order diagnostics for that file
