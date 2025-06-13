// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

import {console} from "forge-std/Script.sol";

contract Precompiler {
    constructor(uint256 index, uint256 gas_target, bool use_long) {
        if (index == 1) {
            run_ecrecover(gas_target);
        } else if (index == 2) {
            run_sha256(gas_target, use_long);
        } else if (index == 3) {
            run_ripemd160(gas_target, use_long);
        } else if (index == 4) {
            run_identity(gas_target, use_long);
        } else if (index == 5) {
            run_modexp(gas_target, use_long);
        } else if (index == 6) {
            run_ecadd(gas_target);
        } else if (index == 7) {
            run_ecmul(gas_target);
        } else if (index == 8) {
            run_ecpairing(gas_target);
        } else if (index == 9) {
            run_blake2f(gas_target);
        } else if (index == 10) {
            // KZG Point Evaluation
            revert("KZG Point Evaluation not implemented, as this precompile is accelerated by the FPVM");
        } else if (index == 0x100) {
            run_p256Verify(gas_target);
        } else if (index == 0x0b) {
            run_g1add(gas_target);
        } else {
            // Invalid index
            revert("Invalid index");
        }
    }

    function hashLongString() public pure returns (string memory) {
        string memory longInput = string(
            abi.encodePacked(
                "This is a long input string for precompile ",
                "and it is being repeated multiple times to increase the size. ",
                "Lorem ipsum dolor sit amet, consectetur adipiscing elit. ",
                "Vivamus luctus urna sed urna ultricies ac tempor dui sagittis. ",
                "In condimentum facilisis porta. Sed nec diam eu diam mattis viverra. ",
                "Nulla fringilla, orci ac euismod semper, magna diam porttitor mauris, ",
                "quis sollicitudin sapien justo in libero. Vestibulum mollis mauris enim. ",
                "Morbi euismod magna ac lorem rutrum elementum. " "This is a long input string for precompile ",
                "and it is being repeated multiple times to increase the size. ",
                "Lorem ipsum dolor sit amet, consectetur adipiscing elit. ",
                "Vivamus luctus urna sed urna ultricies ac tempor dui sagittis. ",
                "In condimentum facilisis porta. Sed nec diam eu diam mattis viverra. ",
                "Nulla fringilla, orci ac euismod semper, magna diam porttitor mauris, ",
                "quis sollicitudin sapien justo in libero. Vestibulum mollis mauris enim. ",
                "Morbi euismod magna ac lorem rutrum elementum. " "This is a long input string for precompile ",
                "and it is being repeated multiple times to increase the size. ",
                "Lorem ipsum dolor sit amet, consectetur adipiscing elit. ",
                "Vivamus luctus urna sed urna ultricies ac tempor dui sagittis. ",
                "In condimentum facilisis porta. Sed nec diam eu diam mattis viverra. ",
                "Nulla fringilla, orci ac euismod semper, magna diam porttitor mauris, ",
                "quis sollicitudin sapien justo in libero. Vestibulum mollis mauris enim. ",
                "Morbi euismod magna ac lorem rutrum elementum. "
            )
        );

        return longInput;
    }

    function run_ecrecover(uint256 gas_target) private {
        uint256 start_gas = gasleft();
        uint256 gas_used = 0;

        uint8 v = 28;
        bytes32 r = 0x9242685bf161793cc25603c231bc2f568eb630ea16aa137d2664ac8038825608;
        bytes32 s = 0x4f8ae3bd7535248d0bd448298cc2e2071e56992d0774dc340c368ae950852ada;
        while (gas_used < gas_target) {
            bytes32 hash = bytes32(gas_used);
            ecrecover(hash, v, r, s);
            gas_used = start_gas - gasleft();
        }
    }

    function run_sha256(uint256 gas_target, bool use_long) private {
        uint256 start_gas = gasleft();
        uint256 gas_used = 0;

        uint256 count = 0;
        while (gas_used < gas_target) {
            count += 1;
            if (use_long) {
                sha256(abi.encodePacked(hashLongString(), gas_used));
            } else {
                sha256(abi.encodePacked(gas_used));
            }
            gas_used = start_gas - gasleft();
        }
        console.log("SHA256 count: %d", count);
    }

    function run_ripemd160(uint256 gas_target, bool use_long) private {
        uint256 start_gas = gasleft();
        uint256 gas_used = 0;

        uint256 count = 0;
        while (gas_used < gas_target) {
            count += 1;
            if (use_long) {
                ripemd160(abi.encodePacked(hashLongString(), gas_used));
            } else {
                ripemd160(abi.encodePacked(gas_used));
            }
            gas_used = start_gas - gasleft();
        }
        console.log("RIPEMD160 count: %d", count);
    }

    function run_identity(uint256 gas_target, bool use_long) private {
        uint256 start_gas = gasleft();
        uint256 gas_used = 0;

        uint256 count = 0;
        while (gas_used < gas_target) {
            count += 1;
            if (use_long) {
                address(4).staticcall(abi.encode(hashLongString(), gas_used));
            } else {
                address(4).staticcall(abi.encode(gas_used));
            }
            gas_used = start_gas - gasleft();
        }
        console.log("Identity count: %d", count);
    }

    function run_modexp(uint256 gas_target, bool use_long) private {
        uint256 start_gas = gasleft();
        uint256 gas_used = 0;

        bytes memory base = "8";
        bytes memory exponent = "9";
        uint256 count = 0;
        while (gas_used < gas_target) {
            count += 1;
            if (use_long) {
                bytes memory modulus = abi.encodePacked(hashLongString(), gas_used);
                address(5).staticcall(
                    abi.encodePacked(base.length, exponent.length, modulus.length, base, exponent, modulus)
                );
            } else {
                bytes memory modulus = abi.encodePacked(gas_used);
                address(5).staticcall(
                    abi.encodePacked(base.length, exponent.length, modulus.length, base, exponent, modulus)
                );
            }
            gas_used = start_gas - gasleft();
        }
        console.log("ModExp count: %d", count);
    }

    function run_ecadd(uint256 gas_target) private {
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

    function run_ecmul(uint256 gas_target) private {
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

    function run_ecpairing(uint256 gas_target) private {
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

    function run_blake2f(uint256 gas_target) private {
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

    function run_p256Verify(uint256 gas_target) private {
        uint256 start_gas = gasleft();
        uint256 gas_used = 0;

        bytes32 x = 0x31a80482dadf89de6302b1988c82c29544c9c07bb910596158f6062517eb089a;
        bytes32 y = 0x2f54c9a0f348752950094d3228d3b940258c75fe2a413cb70baa21dc2e352fc5;
        bytes32 r = 0xe22466e928fdccef0de49e3503d2657d00494a00e764fd437bdafa05f5922b1f;
        bytes32 s = 0xbbb77c6817ccf50748419477e843d5bac67e6a70e97dde5a57e0c983b777e1ad;
        while (gas_used < gas_target) {
            bytes32 hash = bytes32(gas_used);
            (bool ok,) = address(0x100).staticcall(abi.encode(hash, r, s, x, y));
            require(ok, "p256Verify failed");
            gas_used = start_gas - gasleft();
        }
    }

    function run_g1add(uint256 gas_target) private {
        uint256 start_gas = gasleft();
        uint256 gas_used = 0;

        bytes32[4] memory p1;
        p1[0] = 0x0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0f;
        p1[1] = 0xc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb;
        p1[2] = 0x0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4;
        p1[3] = 0xfcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1;

        bytes32[4] memory p2;
        p2[0] = 0x00000000000000000000000000000000112b98340eee2777cc3c14163dea3ec9;
        p2[1] = 0x7977ac3dc5c70da32e6e87578f44912e902ccef9efe28d4a78b8999dfbca9426;
        p2[2] = 0x00000000000000000000000000000000186b28d92356c4dfec4b5201ad099dbd;
        p2[3] = 0xede3781f8998ddf929b4cd7756192185ca7b8f4ef7088f813270ac3d48868a21;

        while (gas_used < gas_target) {
            (bool ok, bytes memory result) = address(11).staticcall(abi.encode(p1, p2));
            require(ok, "G1Add failed");
            p2 = abi.decode(result, (bytes32[4]));
            gas_used = start_gas - gasleft();
        }
    }
}