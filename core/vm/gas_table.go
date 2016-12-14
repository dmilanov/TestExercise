package vm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

func memoryGasCost(mem *Memory, newMemSize *big.Int) *big.Int {
	gas := new(big.Int)
	if newMemSize.Cmp(common.Big0) > 0 {
		newMemSizeWords := toWordSize(newMemSize)

		if newMemSize.Cmp(u256(int64(mem.Len()))) > 0 {
			// be careful reusing variables here when changing.
			// The order has been optimised to reduce allocation
			oldSize := toWordSize(big.NewInt(int64(mem.Len())))
			pow := new(big.Int).Exp(oldSize, common.Big2, Zero)
			linCoef := oldSize.Mul(oldSize, params.MemoryGas)
			quadCoef := new(big.Int).Div(pow, params.QuadCoeffDiv)
			oldTotalFee := new(big.Int).Add(linCoef, quadCoef)

			pow.Exp(newMemSizeWords, common.Big2, Zero)
			linCoef = linCoef.Mul(newMemSizeWords, params.MemoryGas)
			quadCoef = quadCoef.Div(pow, params.QuadCoeffDiv)
			newTotalFee := linCoef.Add(linCoef, quadCoef)

			fee := newTotalFee.Sub(newTotalFee, oldTotalFee)
			gas.Add(gas, fee)
		}
	}
	return gas
}

func makeGenericGasFunc(op OpCode) gasFunc {
	return func(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
		return gasTable[op]
	}
}

func gasCalldataCopy(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	gas := memoryGasCost(mem, memorySize)
	gas.Add(gas, gasTable[CALLDATACOPY])
	words := toWordSize(stack.Back(2))

	return gas.Add(gas, words.Mul(words, params.CopyGas))
}

func gasSStore(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	var (
		y, x = stack.Back(1), stack.Back(0)
		val  = env.StateDB.GetState(contract.Address(), common.BigToHash(x))
	)
	// This checks for 3 scenario's and calculates gas accordingly
	// 1. From a zero-value address to a non-zero value         (NEW VALUE)
	// 2. From a non-zero value address to a zero-value address (DELETE)
	// 3. From a non-zero to a non-zero                         (CHANGE)
	if common.EmptyHash(val) && !common.EmptyHash(common.BigToHash(y)) {
		// 0 => non 0
		return new(big.Int).Set(params.SstoreSetGas)
	} else if !common.EmptyHash(val) && common.EmptyHash(common.BigToHash(y)) {
		env.StateDB.AddRefund(params.SstoreRefundGas)

		return new(big.Int).Set(params.SstoreClearGas)
	} else {
		// non 0 => non 0 (or 0 => 0)
		return new(big.Int).Set(params.SstoreResetGas)
	}
}

func makeGasLog(n uint) gasFunc {
	return func(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
		mSize := stack.Back(1)

		gas := new(big.Int).Add(memoryGasCost(mem, memorySize), params.LogGas)
		gas.Add(gas, new(big.Int).Mul(big.NewInt(int64(n)), params.LogTopicGas))
		gas.Add(gas, new(big.Int).Mul(mSize, params.LogDataGas))
		return gas
	}
}

func gasSha3(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	gas := memoryGasCost(mem, memorySize)
	gas.Add(gas, gasTable[SHA3])
	words := toWordSize(stack.Back(1))
	return gas.Add(gas, words.Mul(words, params.Sha3WordGas))
}

func gasCodeCopy(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	gas := memoryGasCost(mem, memorySize)
	gas.Add(gas, gasTable[CODECOPY])
	words := toWordSize(stack.Back(2))

	return gas.Add(gas, words.Mul(words, params.CopyGas))
}

func gasExtCodeCopy(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	gas := memoryGasCost(mem, memorySize)
	gas.Add(gas, gt.ExtcodeCopy)
	words := toWordSize(stack.Back(3))

	return gas.Add(gas, words.Mul(words, params.CopyGas))
}

func gasMLoad(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	return new(big.Int).Add(gasTable[MLOAD], memoryGasCost(mem, memorySize))
}

func gasMStore8(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	return new(big.Int).Add(gasTable[MSTORE8], memoryGasCost(mem, memorySize))
}

func gasMStore(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	return new(big.Int).Add(gasTable[MSTORE], memoryGasCost(mem, memorySize))
}

func gasCreate(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	return new(big.Int).Add(gasTable[CREATE], memoryGasCost(mem, memorySize))
}

func gasBalance(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	return gt.Balance
}

func gasExtCodeSize(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	return gt.ExtcodeSize
}

func gasSLoad(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	return gt.SLoad
}

func gasExp(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	expByteLen := int64((stack.data[stack.len()-2].BitLen() + 7) / 8)
	gas := big.NewInt(expByteLen)
	gas.Mul(gas, gt.ExpByte)
	return gas.Add(gas, gasTable[EXP])
}

func gasCall(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	gas := new(big.Int).Set(gt.Calls)

	transfersValue := stack.Back(2).BitLen() > 0
	var (
		address = common.BigToAddress(stack.Back(1))
		eip158  = env.ChainConfig().IsEIP158(env.BlockNumber)
	)
	if eip158 {
		if env.StateDB.Empty(address) && transfersValue {
			gas.Add(gas, params.CallNewAccountGas)
		}
	} else if !env.StateDB.Exist(address) {
		gas.Add(gas, params.CallNewAccountGas)
	}
	if transfersValue {
		gas.Add(gas, params.CallValueTransferGas)
	}
	gas.Add(gas, memoryGasCost(mem, memorySize))

	cg := callGas(gt, contract.Gas, gas, stack.data[stack.len()-1])
	// Replace the stack item with the new gas calculation. This means that
	// either the original item is left on the stack or the item is replaced by:
	// (availableGas - gas) * 63 / 64
	// We replace the stack item so that it's available when the opCall instruction is
	// called. This information is otherwise lost due to the dependency on *current*
	// available gas.
	stack.data[stack.len()-1] = cg

	return gas.Add(gas, cg)
}

func gasCallCode(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	gas := new(big.Int).Set(gt.Calls)
	if stack.Back(2).BitLen() > 0 {
		gas.Add(gas, params.CallValueTransferGas)
	}
	gas.Add(gas, memoryGasCost(mem, memorySize))

	cg := callGas(gt, contract.Gas, gas, stack.data[stack.len()-1])
	// Replace the stack item with the new gas calculation. This means that
	// either the original item is left on the stack or the item is replaced by:
	// (availableGas - gas) * 63 / 64
	// We replace the stack item so that it's available when the opCall instruction is
	// called. This information is otherwise lost due to the dependency on *current*
	// available gas.
	stack.data[stack.len()-1] = cg

	return gas.Add(gas, cg)
}

func gasReturn(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	return new(big.Int).Add(gasTable[RETURN], memoryGasCost(mem, memorySize))
}

func gasSuicide(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	gas := new(big.Int)
	// EIP150 homestead gas reprice fork:
	if env.ChainConfig().IsEIP150(env.BlockNumber) {
		gas.Set(gt.Suicide)
		var (
			address = common.BigToAddress(stack.Back(0))
			eip158  = env.ChainConfig().IsEIP158(env.BlockNumber)
		)

		if eip158 {
			// if empty and transfers value
			if env.StateDB.Empty(address) && env.StateDB.GetBalance(contract.Address()).BitLen() > 0 {
				gas.Add(gas, gt.CreateBySuicide)
			}
		} else if !env.StateDB.Exist(address) {
			gas.Add(gas, gt.CreateBySuicide)
		}
	}

	if !env.StateDB.HasSuicided(contract.Address()) {
		env.StateDB.AddRefund(params.SuicideRefundGas)
	}
	return gas
}

func gasDelegateCall(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	gas := new(big.Int).Add(gt.Calls, memoryGasCost(mem, memorySize))

	cg := callGas(gt, contract.Gas, gas, stack.data[stack.len()-1])
	// Replace the stack item with the new gas calculation. This means that
	// either the original item is left on the stack or the item is replaced by:
	// (availableGas - gas) * 63 / 64
	// We replace the stack item so that it's available when the opCall instruction is
	// called.
	stack.data[stack.len()-1] = cg

	return gas.Add(gas, cg)
}

func gasPush(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	return GasFastestStep
}

func gasSwap(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	return GasFastestStep
}

func gasDup(gt params.GasTable, env *Environment, contract *Contract, stack *Stack, mem *Memory, memorySize *big.Int) *big.Int {
	return GasFastestStep
}

var gasTable = [256]*big.Int{
	ADD:          GasFastestStep,
	LT:           GasFastestStep,
	GT:           GasFastestStep,
	SLT:          GasFastestStep,
	SGT:          GasFastestStep,
	EQ:           GasFastestStep,
	ISZERO:       GasFastestStep,
	SUB:          GasFastestStep,
	AND:          GasFastestStep,
	OR:           GasFastestStep,
	XOR:          GasFastestStep,
	NOT:          GasFastestStep,
	BYTE:         GasFastestStep,
	CALLDATALOAD: GasFastestStep,
	CALLDATACOPY: GasFastestStep,
	MLOAD:        GasFastestStep,
	MSTORE:       GasFastestStep,
	MSTORE8:      GasFastestStep,
	CODECOPY:     GasFastestStep,
	MUL:          GasFastStep,
	DIV:          GasFastStep,
	SDIV:         GasFastStep,
	MOD:          GasFastStep,
	SMOD:         GasFastStep,
	SIGNEXTEND:   GasFastStep,
	ADDMOD:       GasMidStep,
	MULMOD:       GasMidStep,
	JUMP:         GasMidStep,
	JUMPI:        GasSlowStep,
	EXP:          GasSlowStep,
	ADDRESS:      GasQuickStep,
	ORIGIN:       GasQuickStep,
	CALLER:       GasQuickStep,
	CALLVALUE:    GasQuickStep,
	CODESIZE:     GasQuickStep,
	GASPRICE:     GasQuickStep,
	COINBASE:     GasQuickStep,
	TIMESTAMP:    GasQuickStep,
	NUMBER:       GasQuickStep,
	CALLDATASIZE: GasQuickStep,
	DIFFICULTY:   GasQuickStep,
	GASLIMIT:     GasQuickStep,
	POP:          GasQuickStep,
	PC:           GasQuickStep,
	MSIZE:        GasQuickStep,
	GAS:          GasQuickStep,
	BLOCKHASH:    GasExtStep,
	BALANCE:      Zero,
	EXTCODESIZE:  Zero,
	EXTCODECOPY:  Zero,
	SLOAD:        params.SloadGas,
	SSTORE:       Zero,
	SHA3:         params.Sha3Gas,
	CREATE:       params.CreateGas,
	CALL:         Zero,
	CALLCODE:     Zero,
	DELEGATECALL: Zero,
	SUICIDE:      Zero,
	JUMPDEST:     params.JumpdestGas,
	RETURN:       Zero,
	PUSH1:        GasFastestStep,
	DUP1:         Zero,
	STOP:         Zero,
}
