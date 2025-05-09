pragma solidity ^0.8.13;

contract Precompile {
    uint256 counter;

    constructor() {
        counter = 1;
    }

    function writer(uint256 gas_target, bytes calldata input2) public {
        uint256 start_gas = gasleft();
        uint256 gas_used = 0;

        while (gas_used < gas_target) {
            assembly {
                sstore(gas_used, gas_used)
            }
            gas_used = start_gas - gasleft();
        }
    }

    function reader(uint256 gas_target, bytes calldata input2) public {
        uint256 start_gas = gasleft();
        uint256 gas_used = 0;

        while (gas_used < gas_target) {
            assembly {
                let junk := sload(gas_used)
            }
            gas_used = start_gas - gasleft();
        }
    }

    function ecadd(uint256 gas_target, bytes calldata input2) public {
        uint256 start_gas = gasleft();
        uint256 gas_used = 0;

        uint256 x1 = 0x030644e72e131a029b85045b68181585d97816a916871ca8d3c208c16d87cfd3;
        uint256 y1 = 0x15ed738c0e0a7c92e7845f96b2ae9c0a68a6a449e3538fc7ff3ebf7a5a18a2c4;
        uint256 x2 = 1;
        uint256 y2 = 2;

        while (gas_used < gas_target) {
            (bool ok, bytes memory result) = address(6).staticcall(abi.encode(x1, y1, x2, y2));
            require(ok, "ECAdd failed");
            (x2, y2) = abi.decode(result, (uint256, uint256));
            gas_used = start_gas - gasleft();
        }
    }

    function increaseCounter(uint256 gas_target, bytes calldata input2) public {
        uint256 start_gas = gasleft();
        uint256 gas_used = 0;
        while (gas_used < gas_target) {
            counter += 1;
            gas_used = start_gas - gasleft();
        }
    }

    function ecmul(uint256 gas_target, bytes calldata input2) public {
        uint256 start_gas = gasleft();
        uint256 gas_used = 0;

        uint256 x1 = 0x030644e72e131a029b85045b68181585d97816a916871ca8d3c208c16d87cfd3;
        uint256 y1 = 0x15ed738c0e0a7c92e7845f96b2ae9c0a68a6a449e3538fc7ff3ebf7a5a18a2c4;
        uint256 scalar = 2;

        while (gas_used < gas_target) {
            (bool ok, bytes memory result) = address(7).staticcall(abi.encode(x1, y1, scalar));
            require(ok, "ECMul failed");
            (x1, y1) = abi.decode(result, (uint256, uint256));
            gas_used = start_gas - gasleft();
        }
    }

    function ecpairing(uint256 gas_target, bytes calldata input2) public {
        uint256 start_gas = gasleft();
        uint256 gas_used = 0;

        uint256[6] memory input = [
            0x2cf44499d5d27bb186308b7af7af02ac5bc9eeb6a3d147c186b21fb1b76e18da,
            0x2c0f001f52110ccfe69108924926e45f0b0c868df0e7bde1fe16d3242dc715f6,
            0x1fb19bb476f6b9e44e2a32234da8212f61cd63919354bc06aef31e3cfaff3ebc,
            0x22606845ff186793914e03e21df544c34ffe2f2f3504de8a79d9159eca2d98d9,
            0x2bd368e28381e8eccb5fa81fc26cf3f048eea9abfdd85d7ed3ab3698d63e4f90,
            0x2fe02e47887507adf0ff1743cbac6ba291e66f59be6bd763950bb16041a0a85e
        ];
        while (gas_used < gas_target) {
            (bool ok, bytes memory result) = address(8).staticcall(abi.encode(input));
            require(ok, "ECPairing failed");
            // Use ECAdd to create new points
            (ok, result) = address(6).staticcall(abi.encode(input[0], input[1], 1, 2));
            require(ok, "ECAdd failed");
            (input[0], input[1]) = abi.decode(result, (uint256, uint256));
            gas_used = start_gas - gasleft();
        }
    }

    function run_blake2f(uint256 gas_target, bytes calldata input2) public {
        uint256 start_gas = gasleft();
        uint256 gas_used = 0;

        // Blake2f
        bytes32[2] memory h;
        h[0] = 0x48c9bdf267e6096a3ba7ca8485ae67bb2bf894fe72f36e3cf1361d5f3af54fa5;
        h[1] = 0xd182e6ad7f520e511f6c3e2b8c68059b6bbd41fbabd9831f79217e1319cde05b;

        bytes32[4] memory m;
        m[0] = 0x6162630000000000000000000000000000000000000000000000000000000000;
        m[1] = 0x0000000000000000000000000000000000000000000000000000000000000000;
        m[2] = 0x0000000000000000000000000000000000000000000000000000000000000000;
        m[3] = 0x0000000000000000000000000000000000000000000000000000000000000000;

        bytes8[2] memory t;
        t[0] = 0x0300000000000000;
        t[1] = 0x0000000000000000;

        bool f = true;

        while (gas_used < gas_target) {
            uint32 rounds = uint32(gas_used / 100);

            (bool ok,) =
                address(9).staticcall(abi.encodePacked(rounds, h[0], h[1], m[0], m[1], m[2], m[3], t[0], t[1], f));
            require(ok, "Blake2f failed");
            gas_used = start_gas - gasleft();
        }
    }


    function getResult() public view returns (uint256) {
        return counter;
    }
}