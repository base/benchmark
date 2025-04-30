# Contract Transaction Payload Documentation

TL;DR: A testing framework that lets you benchmark EVM chain performance using custom smart contracts. Instead of writing complex test scenarios, just compile your contract locally and provide the bytecode in the config. All test functions must accept (uint256, bytes) parameters and include a getResult() function. Configure calls per block, function signature, and inputs through YAML files.

## 1. Purpose
The Contract Transaction Payload system provides a flexible and efficient way to test complex transaction scenarios. Instead of requiring users to manually craft transactions or implement heavyweight testing solutions, this system allows users to:

- Provide smart contract bytecode that adheres to the interface described below. 
- Execute standardized functions on this contract with configurable inputs.
- Measure and analyze the performance of the system compared to other contracts or transaction payload types.

## 2. Limitations
- All test functions must conform to a standardized interface with two parameters:
  - `uint256`: First input parameter for numeric values
  - `bytes calldata`: Second input parameter for arbitrary data
- Contracts must implement a `getResult()` function that returns a `uint256`
- Contract compilation by the user. The contract is not checked into the repository. This is an upcoming improvement we will make in the near future.

## 3. Enforced Smart Contract Interface
```solidity
contract Contract {
    constructor() {}

    // All test functions must follow this parameter pattern
    function function1(uint256 input1, bytes calldata input2) public {
        // Implementation specific to test case
    }

    function function2(uint256 input1, bytes calldata input2) public {
        // Implementation specific to test case
    }

    // Required result retrieval function
    function getResult() public view returns (uint256) {
        // Return test-specific result
    }
}
```

## 4. Example
Here's an example configuration yaml that that demonstrates a contract test scenario:

```yaml
- name: Test custom contract behavior on Geth and Reth
  description: Geth Execution Speed
  benchmark:
    - sequencer
  variables:
    - type: transaction_workload
      values:
        - contract:100:function1(uint256,bytes):20000000:0x1234...[hex calldata]:[contract bytecode]
    - type: node_type
      values:
        - geth
        - reth
    - type: num_blocks
      value: 200
    - type: gas_limit
      value: 2000000000
```

### Configuration Format
The contract configuration follows this pattern:
contract:callsPerBlock:functionSignature:input1:input2:bytecode where:
- contract: test type (does not change)
- callsPerBlock: Number of function calls per block
- functionSignature: Function name with parameter types (e.g., "function1(uint256,bytes)")
- input1: uint256 value
- input2: hex encoded bytes calldata
- bytecode: Compiled contract bytecode

This example shows:
- Testing function1 (some computationally intensive code)
- 100 calls per block
- Specific function signature with configured inputs
- Pre-compiled contract bytecode
- Test execution across 200 blocks
- High gas limit to accommodate complex operations

## 5. Future Improvements
Potential enhancements to consider:
- Support for more complex parameter types
- Contract tracked in source control (included in repository) with solidity compilation automated
- Parallel execution patterns with multiple functions
- Dynamic input generation
- Support for contract-to-contract interactions
- More flexible configuration options
