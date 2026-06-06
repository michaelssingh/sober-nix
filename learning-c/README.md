# Learning C Programming

Welcome to your C programming sandbox. C is a compiled, low-level language that exposes you directly to memory management, machine structures, and minimal abstractions.

## Getting Started

### 1. Enter the Environment
Start a Nix development shell to load the C toolchain (`gcc`, `gdb`, `valgrind`, `make`):
```bash
nix-shell shell.nix
```

### 2. Compile and Run
Navigate into the `01-hello` directory and run the compilation target:
```bash
cd 01-hello
make
./hello
```
To clean up compile artifacts, run:
```bash
make clean
```

---

## 3 Core C Concepts for Beginners

### I. The Compilation Pipeline
Unlike interpreted languages (like Python or JavaScript), C source code is compiled directly into native machine code.
When you run `gcc -Wall -Wextra -O2 -g -o hello hello.c`:
1. **Preprocessing**: Evaluates directives starting with `#` (like `#include <stdio.h>`), copying external declarations directly into the file.
2. **Compilation**: Translates C code into assembly instructions.
3. **Assembly**: Translates assembly code into binary object files (`hello.o`).
4. **Linking**: Links the object files with libraries (such as the standard C library for `printf`) to produce the final executable.

### II. Pointers (Direct Memory Addresses)
A **pointer** is a variable that stores the memory address of another variable. Pointers are denoted by `*` during declaration, and the address-of operator `&` is used to retrieve addresses.

```c
int num = 42;
int *p = &num; // p now stores the memory address of num

printf("Value of num: %d\n", num);    // Prints 42
printf("Address of num: %p\n", &num); // Prints the hex memory address
printf("Value at address: %d\n", *p); // Dereferencing p: prints 42
```

### III. Manual Memory Management
C does not have a Garbage Collector. Memory allocated on the **Stack** (like local variables inside functions) is automatically freed when the function exits. However, memory allocated on the **Heap** must be manually allocated and freed.

```c
// Allocate space for 10 integers on the Heap
int *arr = malloc(10 * sizeof(int));

if (arr == NULL) {
    // Check if allocation succeeded
    return 1; 
}

// ... use the array ...

// Always free your allocated memory!
free(arr);
```

---

## Next Recommended Steps
1. Play with `01-hello/hello.c` by modifying the text and outputting different types of variables.
2. Learn about standard input/output (`scanf`, `fgets`).
3. Experiment with simple control statements (`if`, `loops`) and functions.
