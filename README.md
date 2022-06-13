

![Atomic](https://storage.googleapis.com/kyoto.catchpole.net/atomic-logo.png "Atomic Logo")

# Atomic

A strongly typed concurrent programming language with solution search and dynamic processor profiles.

### Project Status

It is nowhere near usable, but it does do this...

- Minimal viable parser with basic language features.
- AST processing with folding and pruning.
- Loads a profile for 64 bit ARM `aarch64` registers and instructions.
- An `object compositing` operation (copies fields from source objects to a target) for demo purposes.
- This operation utilises simple instruction search, register allocation and lookup and code emitting.
- Generates linkable objects.
  - Mach-o for MacOS on M1 Processors.
  - ELF for Linux on Raspberry Pi 3 onwards etc.
- Links to an example `main()` function creating an executable (see example below).

Therefor any grand claims beyond this point is just a wish list...

## Solution Search

What do we mean by this?

Fundamentally, programs retrieve data from memory (or some kind of input),
possibly modify, combine or re-arrange data, maybe make branching decisions from the results,
and then maybe store it away again.

The idea behind Atomic is to abstract out the fundamentals of these operations and
use search algorithms to determine the machine code instructions required to compile the code to do the job.

Rather than an Abstract Syntax Tree which represents simple discrete operations, it models the expected outputs
of any operation, allowing the compiler to search back to find the required inputs and permutations of those inputs. 

## Atomic Language

The Atomic Language has strong language level support for:

- Collections, data structures and data models.

## Atomic Functions

Atomic functions are characterised by:

- Highly scoped, only receiving inputs and emitting outputs via pointers passed in parameter registers.
- Non-blocking, I/O performed external to atomic functions with data marshalled via the function's inputs and outputs. 
- Can only call other atomic functions resolved via those structures.
- All constants that can not be compiled as inline immediate values are passed via those structures.
This allows for various kinds of data and function substitution that static linking might not. 
- Have no program counter relative references to locations outside the function.

The benefits include:

- Highly retargetable, scaling from embedded systems to large servers dynamically loading or compiling functions.
- Easier to reason about function behavior, test case assertions etc.

## Slippery Functions

Atomic functions can also be slippery functions.

A slippery function leverages the `aarch64` use of a `link` register and `return` instructions being simply a
`branch` to an arbitrary location `register` (not necessarily the `link` register). 
This allows functions to easily return to different locations, as long as the alternate path provide a compatible stack frame.

The benefits include:

- Simplifies exception like handling, avoiding result code creation and checking.
- Efficient "happy path" and exception path handling.
- Case or conditional processing can be translated to slippery `return` tables (effectively jump tables).
- A structured mechanism to cancel threads by redirecting return paths.

## Slippery Streams

Slippery Streams are a mechanism to access sequences of types from bits up to structures, as single, fixed
or dynamic length. Their purpose is to simplify access to data, scaling from small, single instance types
up to large stream or file based data, with any required memory management or I/O being performed transparently.

Slippery Streams are effectively the mechanisms to access single instance types, fixed width arrays,
dynamic arrays and I/O streams (files, network connections etc) combined into one. Code attempting to
operate on the streams in various ways determines how they are represented by the compiler.

## Dynamic Processor Profiles

Atomic generates machine code based on processor profiles which are dynamically defined in text files.
The baseline profile is for 64 bit ARM (`arm64` / `aarch64`).
The current implementation assumes a load-store architecture, a 32 bit instruction size and certain features of `aarch64`.

The benefits of processor profiles are:

- Adding CPU support only requires creating a new profile (maybe).
- Encourages the use of solution search in the compiler, rather than hard coded solutions for each CPU.
- Simple to make platform dependant changes (eg. reserving register `x18` on Mac OS but not Linux).
- Removes the need for assembler output or an inbuilt platform specific assembler.

Atomic currently emits:

- Mach-o objects for MacOS on M1 Processors
- ELF objects for Linux on Raspberry Pi 3 onwards etc.

These can be linked against a `main()` dispatcher to produce executables.

## Compiler Features

- Concurrent loading, parsing and compiling of functions, based on the host CPU core count.

## Example output

Running..

```
airline.sh
```

This simple function populates a boarding pass with values from the available inputs...

```c
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

```asm
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

Then linked with this test code written in C, which sets up the inputs and calls `makeBoardingPass()`

```c
    char* passenger[] = { "Slippery Seal" };
    char* airport[] = { "Adelaide", "ADL"};
    char* gate[] = { "Gate 1" };
    char* flight[] = { "AA001" };
    char* boardingPass[4];

    makeBoardingPass(&passenger, &airport, &gate, &flight, &boardingPass);
```

The resulting `BoardingPass`...

```asm
passengerName: Slippery Seal
airportCode: ADL
flightNumber: AA001
gateNumber: Gate 1
```

Not rocket science, but it is an end to end demo.

## Infrequently Asked Questions

Isn't search going to be slow?

- Perhaps. But common solutions could be pre-computed or cached at various stages of compilation.
It is also hoped that a simpler compiler design will compensate for search time.
AST processing should be limited to efficient decent and ascent phases. 
There is no intermediate representation (IR) or complex distilling of output. 

Aren't programming languages and compilers a solved problem? Surely The Terminator will run on Python and SQL?

- Everyone knows the Cyberdyne Systems Model 101 Series 800 Terminator runs on 6502 assembler. And don't call me Shirley.

