// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import {ERC20} from "@openzeppelin/contracts/token/ERC20/ERC20.sol";

contract ERC20Transfer is ERC20 {
    uint256 counter;
    uint256 private nonce = 0;

    constructor() ERC20("Token", "TKN") {
        _mint(msg.sender, 10000000 * 10 ** decimals());
        counter = 5;
    }

    function generateRandomAddress() public returns (address) {
        nonce++;

        bytes32 hash = keccak256(
            abi.encodePacked(block.timestamp, msg.sender, nonce)
        );

        return address(uint160(uint256(hash)));
    }

    function increaseCounter(uint256 gas_target, bytes calldata input2) public {
        counter += 1;
    }

    function moveErc20(uint256 gas_target, bytes calldata input2) public {
        uint256 start_gas = gasleft();
        uint256 gas_used = 0;

        address to = generateRandomAddress();
        counter += 1;
        while (gas_used < gas_target) {
            transfer(to, 1 wei);
            gas_used = start_gas - gasleft();
            counter++;
        }
    }

    function getResult() public view returns (uint256) {
        return counter;
    }
}