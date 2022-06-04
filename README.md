
# Atomic

A strongly typed concurrent programming language with solution search and dynamic processor profiles.

### Status

- Minimal viable parser with some language features
- AST processing and optimisation
- Loads a profile for 64 bit ARM `aarch64` registers and instructions
- An `object compositing` operation
- Generates Mac OS mach-o objects ready for linking (Linux EFL in development)
- Links to example `main()` function creating an executable (see example below)

## Atomic Language

The Atomic Language has strong language level support for:

- Collections, data structures and data models.

## Type Streams

Type streams are a mechanism to access sequences of types from bits up to structures, as single, fixed
or dynamic length.

## Atomic Functions

Atomic functions are characterised by:

- Highly scoped, only receiving inputs and emitting outputs via pointers passed in parameter registers.
- Non-blocking, I/O performed external to atomic functions with data marshalled via the function's inputs and outputs. 
- Can only call other atomic functions resolved via those structures.
- All constants that can not be compiled as inline immediate values are passed via those structures.
This allows for various kinds of data and function substitution that static linking might not. 
- Have no program counter relative references to locations outside the function.

The benefits of this include:

- Highly retargetable, scaling from embedded systems to large servers dynamically loading or compiling functions.
- Easier to reason about function behavior, test case assertions etc.

## Slippery Functions

Atomic functions can also be slippery functions.

A slippery function leverages the `aarch64` use of a `link` register and `return` instructions being simply a
`branch` to an arbitrary location `register` (not necessarily the `link` register). 
This allows functions to easily return to different locations, as long as the new path provide a compatible stack frame.

The advantage of slippery functions are:

- simple exception like handling
- avoids result code creation and checking
- case or conditional processing can be translated to slippery `return` tables (effectively jump tables)
- fast efficient "happy path" and exception path handling

## Dynamic Processor Profiles

Atomic generates machine code based on processor profiles which are dynamically defined in configuration files.
The baseline profile is for 64 bit ARM (`arm64` / `aarch64`).
The current implementation assumes a load-store architecture, a 32 bit instruction size and certain features of `aarch64`.

The benefits of processor profiles are:

- CPU support only requires creating a new profile (maybe)
- encourages the use of solution search in the compiler, rather than hard coded solutions for each CPU
- simple to make platform dependant changes (eg. reserving register `x18` on Mac OS but not Linux)
- removes the need for assembler output or an inbuilt platform specific assembler

Atomic currently emits Mach64 objects compatible with the Mac OS linker for M1 processors. 
These can be linked against a main dispatcher to produce executables.

ELF object generation for linux (eg. 64 bit Raspberry Pi) is in development (see `elf.go`)

## Compiler Features

- Concurrent loading, parsing and compiling of functions, based on the host CPU core count.

## Example output

This simple function populates a boarding pass with values from the available inputs...

```
function: makeBoardingPass {
    > passenger
    > airport
    > gate
    > flight
    > boardingPass

    + boardingPass
}
```

The generated machine code for `makeBoardingPass()` as viewed with `objdump`

```
airline.o:	file format mach-o arm64

Disassembly of section __TEXT,__text:

0000000000000000 <_makeBoardingPass>:
0: 05 00 40 f9  	ldr	x5, [x0]
4: 85 00 00 f9  	str	x5, [x4]
8: 25 04 40 f9  	ldr	x5, [x1, #8]
c: 85 04 00 f9  	str	x5, [x4, #8]
10: 65 00 40 f9  	ldr	x5, [x3]
14: 85 08 00 f9  	str	x5, [x4, #16]
18: 45 00 40 f9  	ldr	x5, [x2]
1c: 85 0c 00 f9  	str	x5, [x4, #24]

0000000000000020 <exit_makeBoardingPass>:
20: c0 03 5f d6  	ret
```

Then linked with this test code written in C, which populates the inputs and calls `makeBoardingPass()`

```
    char* passenger[] = { "Slippery Seal" };
    char* airport[] = { "Adelaide", "ADL"};
    char* gate[] = { "Gate 1" };
    char* flight[] = { "AA001" };
    char* boardingPass[] = {0,0,0,0};

    makeBoardingPass(&passenger, &airport, &gate, &flight, &boardingPass);
```

The resulting `BoardingPass`...

```
passengerName: Slippery Seal
airportCode: ADL
flightNumber: AA001
gateNumber: Gate 1
```

Not rocket science but it is an end to end demo.
