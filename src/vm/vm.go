package vm

import (
	"fmt"

	"github.com/vdchnsk/qrk/src/code"
	"github.com/vdchnsk/qrk/src/compiler"
	"github.com/vdchnsk/qrk/src/object"
	"github.com/vdchnsk/qrk/src/utils"
)

const StackSize = 2048

type VM struct {
	constants    []object.Object
	instructions code.Instructions

	stack []object.Object

	// Always points to the first free slot on the stack
	stackPointer int
}

func NewVm(bytecode *compiler.Bytecode) *VM {
	stack := make([]object.Object, StackSize)

	return &VM{
		instructions: bytecode.Instructions,
		constants:    bytecode.Constants,

		stack:        stack,
		stackPointer: 0,
	}
}

func (vm *VM) Run() error {
	for instructionPointer := 0; instructionPointer < len(vm.instructions); instructionPointer++ {
		instructionByte := vm.instructions[instructionPointer]

		opcode := code.Opcode(instructionByte)

		switch opcode {
		case code.OpConstant:
			constantIndex := utils.ReadUint16(vm.instructions[instructionPointer+1:])
			readBytes := 2
			instructionPointer += readBytes

			err := vm.stackPush(vm.constants[constantIndex])

			if err != nil {
				return err
			}

		case code.OpAdd:
			right := vm.stackPop()
			left := vm.stackPop()

			// * we assume that both right and left are integers
			rightValue := right.(*object.Integer).Value
			leftValue := left.(*object.Integer).Value

			result := leftValue + rightValue
			vm.stackPush(&object.Integer{Value: result})

		case code.OpPop:
			vm.stackPop()
		}
	}

	return nil
}

func (vm *VM) StackTop() object.Object {
	if vm.stackPointer == 0 {
		return nil
	}

	topStackElem := vm.stack[vm.stackPointer-1]

	return topStackElem
}

// Use only for tests
func (vm *VM) LastPoppedStackElem() object.Object {
	// since, when popping we only decrement the pointer and not actually overriding top of the stack with nil
	// we can assume that after performing a pop() operation stack pointer will be pointing to the element that was just popped
	return vm.stack[vm.stackPointer]
}

func (vm *VM) stackPush(elem object.Object) error {
	isStackOverflow := vm.stackPointer >= StackSize
	if isStackOverflow {
		return fmt.Errorf("stack overflow")
	}

	firstStackFreeSlot := vm.stackPointer

	vm.stack[firstStackFreeSlot] = elem
	vm.stackPointer++

	return nil
}

func (vm *VM) stackPop() object.Object {
	topStackElem := vm.stack[vm.stackPointer-1]

	vm.stackPointer--

	return topStackElem
}
