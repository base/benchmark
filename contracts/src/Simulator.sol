// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

struct PrecompileConfig {
    address precompile_address;
    uint256 num_calls;
}

struct SimulatorConfig {
    uint160 load_accounts;
    uint160 update_accounts;
    uint160 create_accounts;
    uint256 load_storage;
    uint256 update_storage;
    uint256 delete_storage;
    uint256 create_storage;
    PrecompileConfig[] precompiles;
}

interface IExtsload {
    function extsload(bytes32 slot) external view returns (bytes32);
}

contract Simulator {
    uint256 constant storage_chunk_size = 100;
    uint160 constant address_chunk_size = 100;
    uint256 constant safe_offset = 2 << 128;
    uint160 constant safe_address_offset = 2 << 16;

    mapping(uint256 => uint256) storage_slots;
    uint256 public num_storage_initialized = safe_offset;
    uint160 public num_address_initialized = safe_address_offset;
    uint256 public num_storage_deleted = safe_offset;

    // first storage slot with a value
    uint256 current_storage_slot_index = safe_offset;
    uint160 current_address_index = safe_address_offset;

    constructor(uint160 offset) payable {
        // runtime offset allows us to run multiple simulators sequentially without conflicts
        current_address_index += offset;
        num_address_initialized += offset;
    }

    function initialize_storage_chunk() public {
        uint256 start_index = num_storage_initialized;
        uint256 end_index = num_storage_initialized + storage_chunk_size;

        for (uint256 i = start_index; i < end_index; i++) {
            storage_slots[i] = i;
        }
        num_storage_initialized += storage_chunk_size;
    }

    function initialize_address_chunk() public {
        uint160 start_index = num_address_initialized;
        uint160 end_index = num_address_initialized + address_chunk_size;

        // ignore return value
        bool success;
        for (uint160 i = start_index; i < end_index; i++) {
            success = payable(address(i)).send(1);
        }
        num_address_initialized += address_chunk_size;
    }

    function num_storage_slots_needed(SimulatorConfig calldata config) public view returns (uint256) {
        return current_storage_slot_index +
                config.load_storage +
                config.update_storage;
    }

    function num_accounts_needed(SimulatorConfig calldata config) public view returns (uint160) {
        return current_address_index +
                config.load_accounts +
                config.update_accounts;
    }

    function run(SimulatorConfig calldata config) public {
        require(
            current_storage_slot_index +
                config.load_storage +
                config.update_storage <=
                num_storage_initialized,
            "Not enough storage slots to load/update"
        );

        require(
            current_address_index +
                config.load_accounts +
                config.update_accounts <=
                num_address_initialized,
            "Not enough accounts to load/update"
        );

        // load storage slots using SLOAD in a loop. Ensure we're loading a unique storage slot each time.
        uint256 total = 0;
        for (
            uint256 i = current_storage_slot_index;
            i < current_storage_slot_index + config.load_storage;
            i++
        ) {
            total += uint256(IExtsload(address(0x498581fF718922c3f8e6A244956aF099B2652b2b)).extsload(bytes32(i)));
        }
        current_storage_slot_index += config.load_storage;

        // starting from current_storage_slot_index, update existing storage slots in a loop (using SSTORE)
        for (
            uint256 i = current_storage_slot_index;
            i < current_storage_slot_index + config.update_storage;
            i++
        ) {
            assembly {
                sstore(i, i)
            }
        }
        current_storage_slot_index += config.update_storage;

        // starting from num_storage_initialized, create new storage slots in a loop (using SSTORE)
        for (
            uint256 i = num_storage_initialized;
            i < num_storage_initialized + config.create_storage;
            i++
        ) {
            assembly {
                sstore(i, i)
            }
        }
        num_storage_initialized += config.create_storage;

        // starting from 0, delete storage slots in a loop (using SSTORE)
        for (
            uint256 i = num_storage_deleted;
            i < num_storage_deleted + config.delete_storage;
            i++
        ) {
            assembly {
                sstore(i, 0)
            }
        }
        num_storage_deleted += config.delete_storage;

        // create new accounts in a loop (using CREATE)
        for (uint160 i = num_address_initialized; i < num_address_initialized + config.create_accounts; i++) {
            payable(address(i)).send(1);
        }
        num_address_initialized += config.create_accounts;

        // load existing accounts in a loop
        for (uint160 i = current_address_index; i < current_address_index + config.load_accounts; i++) {
            assembly {
                pop(balance(i))
            }
        }
        current_address_index += config.load_accounts;

        // update existing accounts in a loop
        for (uint160 i = current_address_index; i < current_address_index + config.update_accounts; i++) {
            payable(address(i)).send(1);
        }
        current_address_index += config.update_accounts;

        for (uint256 i = 0; i < config.precompiles.length; i++) {
            run_precompile(
                config.precompiles[i].precompile_address,
                config.precompiles[i].num_calls
            );
        }
    }

    function run_precompile(
        address precompile_address,
        uint256 num_calls
    ) private {
        if (precompile_address == address(1)) {
            run_ecrecover(num_calls);
        } else if (precompile_address == address(2)) {
            run_sha256(num_calls, true);
        } else if (precompile_address == address(3)) {
            run_ripemd160(num_calls, true);
        } else if (precompile_address == address(4)) {
            run_identity(num_calls, true);
        } else if (precompile_address == address(5)) {
            run_modexp(num_calls, true);
        } else if (precompile_address == address(6)) {
            run_ecadd(num_calls);
        } else if (precompile_address == address(7)) {
            run_ecmul(num_calls);
        } else if (precompile_address == address(8)) {
            run_ecpairing(num_calls);
        } else if (precompile_address == address(9)) {
            run_blake2f(num_calls);
        } else if (precompile_address == address(0x100)) {
            run_p256Verify(num_calls);
        } else if (precompile_address == address(0x0b)) {
            run_g1add(num_calls);
        } else if (precompile_address == address(0x0c)) {
            run_g1msm(num_calls);
        } else if (precompile_address == address(0x0d)) {
            run_g2add(num_calls);
        } else if (precompile_address == address(0x0e)) {
            run_g2msm(num_calls);
        } else if (precompile_address == address(0x0f)) {
            run_bls_pairing(num_calls);
        } else if (precompile_address == address(0x10)) {
            run_map_g1(num_calls);
        } else if (precompile_address == address(0x11)) {
            run_map_g2(num_calls);
        } else {
            revert("Invalid precompile address");
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

    function run_ecrecover(uint256 num_iterations) private {
        uint8 v = 28;
        bytes32 r = 0x9242685bf161793cc25603c231bc2f568eb630ea16aa137d2664ac8038825608;
        bytes32 s = 0x4f8ae3bd7535248d0bd448298cc2e2071e56992d0774dc340c368ae950852ada;

        for (uint256 i = 0; i < num_iterations; i++) {
            bytes32 hash = bytes32(i);
            ecrecover(hash, v, r, s);
        }
    }

    function run_sha256(uint256 num_iterations, bool use_long) private {
        for (uint256 i = 0; i < num_iterations; i++) {
            if (use_long) {
                sha256(abi.encodePacked(hashLongString(), i));
            } else {
                sha256(abi.encodePacked(i));
            }
        }
    }

    function run_ripemd160(uint256 num_iterations, bool use_long) private {
        for (uint256 i = 0; i < num_iterations; i++) {
            if (use_long) {
                ripemd160(abi.encodePacked(hashLongString(), i));
            } else {
                ripemd160(abi.encodePacked(i));
            }
        }
    }

    function run_identity(uint256 num_iterations, bool use_long) private {
        for (uint256 i = 0; i < num_iterations; i++) {
            if (use_long) {
                address(4).staticcall(abi.encode(hashLongString(), i));
            } else {
                address(4).staticcall(abi.encode(i));
            }
        }
    }

    function run_modexp(uint256 num_iterations, bool use_long) private {
        bytes memory base = "8";
        bytes memory exponent = "9";
        
        for (uint256 i = 0; i < num_iterations; i++) {
            if (use_long) {
                bytes memory modulus = abi.encodePacked(hashLongString(), i);
                address(5).staticcall(
                    abi.encodePacked(base.length, exponent.length, modulus.length, base, exponent, modulus)
                );
            } else {
                bytes memory modulus = abi.encodePacked(i);
                address(5).staticcall(
                    abi.encodePacked(base.length, exponent.length, modulus.length, base, exponent, modulus)
                );
            }
        }
    }

    function run_ecadd(uint256 num_iterations) private {
        uint256 x1 = 0x030644e72e131a029b85045b68181585d97816a916871ca8d3c208c16d87cfd3;
        uint256 y1 = 0x15ed738c0e0a7c92e7845f96b2ae9c0a68a6a449e3538fc7ff3ebf7a5a18a2c4;
        uint256 x2 = 1;
        uint256 y2 = 2;

        for (uint256 i = 0; i < num_iterations; i++) {
            (bool ok, bytes memory result) = address(6).staticcall(abi.encode(x1, y1, x2, y2));
            require(ok, "ECAdd failed");
            (x2, y2) = abi.decode(result, (uint256, uint256));
        }
    }

    function run_ecmul(uint256 num_iterations) private {
        uint256 x1 = 0x030644e72e131a029b85045b68181585d97816a916871ca8d3c208c16d87cfd3;
        uint256 y1 = 0x15ed738c0e0a7c92e7845f96b2ae9c0a68a6a449e3538fc7ff3ebf7a5a18a2c4;
        uint256 scalar = 2;

        for (uint256 i = 0; i < num_iterations; i++) {
            (bool ok, bytes memory result) = address(7).staticcall(abi.encode(x1, y1, scalar));
            require(ok, "ECMul failed");
            (x1, y1) = abi.decode(result, (uint256, uint256));
        }
    }

    function run_ecpairing(uint256 num_iterations) private {
        uint256[6] memory input = [
            0x2cf44499d5d27bb186308b7af7af02ac5bc9eeb6a3d147c186b21fb1b76e18da,
            0x2c0f001f52110ccfe69108924926e45f0b0c868df0e7bde1fe16d3242dc715f6,
            0x1fb19bb476f6b9e44e2a32234da8212f61cd63919354bc06aef31e3cfaff3ebc,
            0x22606845ff186793914e03e21df544c34ffe2f2f3504de8a79d9159eca2d98d9,
            0x2bd368e28381e8eccb5fa81fc26cf3f048eea9abfdd85d7ed3ab3698d63e4f90,
            0x2fe02e47887507adf0ff1743cbac6ba291e66f59be6bd763950bb16041a0a85e
        ];

        for (uint256 i = 0; i < num_iterations; i++) {
            (bool ok, bytes memory result) = address(8).staticcall(abi.encode(input));
            require(ok, "ECPairing failed");
            // Use ECAdd to create new points
            (ok, result) = address(6).staticcall(abi.encode(input[0], input[1], 1, 2));
            require(ok, "ECAdd failed");
            (input[0], input[1]) = abi.decode(result, (uint256, uint256));
        }
    }

    function run_blake2f(uint256 num_iterations) private {
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

        for (uint256 i = 0; i < num_iterations; i++) {
            uint32 rounds = uint32(i);
            (bool ok,) = address(9).staticcall(abi.encodePacked(rounds, h[0], h[1], m[0], m[1], m[2], m[3], t[0], t[1], f));
            require(ok, "Blake2f failed");
        }
    }

    function run_p256Verify(uint256 num_iterations) private {
        bytes32 x = 0x31a80482dadf89de6302b1988c82c29544c9c07bb910596158f6062517eb089a;
        bytes32 y = 0x2f54c9a0f348752950094d3228d3b940258c75fe2a413cb70baa21dc2e352fc5;
        bytes32 r = 0xe22466e928fdccef0de49e3503d2657d00494a00e764fd437bdafa05f5922b1f;
        bytes32 s = 0xbbb77c6817ccf50748419477e843d5bac67e6a70e97dde5a57e0c983b777e1ad;

        for (uint256 i = 0; i < num_iterations; i++) {
            bytes32 hash = bytes32(i);
            (bool ok,) = address(0x100).staticcall(abi.encode(hash, r, s, x, y));
            require(ok, "p256Verify failed");
        }
    }

    function run_g1add(uint256 num_iterations) private {
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

        for (uint256 i = 0; i < num_iterations; i++) {
            (bool ok, bytes memory result) = address(11).staticcall(abi.encode(p1, p2));
            require(ok, "G1Add failed");
            p2 = abi.decode(result, (bytes32[4]));
        }
    }

    function run_g1msm(uint256 num_iterations) private {
        bytes32[4] memory p1;
        p1[0] = 0x0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0f;
        p1[1] = 0xc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb;
        p1[2] = 0x0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4;
        p1[3] = 0xfcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1;

        bytes32 scalar = 0xe22466e928fdccef0de49e3503d2657d00494a00e764fd437bdafa05f5922b1f;

        for (uint256 i = 0; i < num_iterations; i++) {
            (bool ok, bytes memory result) = address(12).staticcall(abi.encode(p1, scalar));
            require(ok, "G1Add failed");
            p1 = abi.decode(result, (bytes32[4]));
        }
    }

    function run_g2add(uint256 num_iterations) private {
        bytes32[8] memory p1;
        p1[0] = 0x00000000000000000000000000000000103121a2ceaae586d240843a39896732;
        p1[1] = 0x5f8eb5a93e8fea99b62b9f88d8556c80dd726a4b30e84a36eeabaf3592937f27;
        p1[2] = 0x00000000000000000000000000000000086b990f3da2aeac0a36143b7d7c8244;
        p1[3] = 0x28215140db1bb859338764cb58458f081d92664f9053b50b3fbd2e4723121b68;
        p1[4] = 0x000000000000000000000000000000000f9e7ba9a86a8f7624aa2b42dcc8772e;
        p1[5] = 0x1af4ae115685e60abc2c9b90242167acef3d0be4050bf935eed7c3b6fc7ba77e;
        p1[6] = 0x000000000000000000000000000000000d22c3652d0dc6f0fc9316e14268477c;
        p1[7] = 0x2049ef772e852108d269d9c38dba1d4802e8dae479818184c08f9a569d878451;

        bytes32[8] memory p2;
        p2[0] = 0x00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051;
        p2[1] = 0xc6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8;
        p2[2] = 0x0000000000000000000000000000000013e02b6052719f607dacd3a088274f65;
        p2[3] = 0x596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e;
        p2[4] = 0x000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351a;
        p2[5] = 0xadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801;
        p2[6] = 0x000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99;
        p2[7] = 0xcb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be;

        for (uint256 i = 0; i < num_iterations; i++) {
            (bool ok, bytes memory result) = address(13).staticcall(abi.encode(p1, p2));
            require(ok, "G2Add failed");
            p2 = abi.decode(result, (bytes32[8]));
        }
    }


    function run_g2msm(uint256 num_iterations) private {
        bytes32[8] memory p1;
        p1[0] = 0x00000000000000000000000000000000103121a2ceaae586d240843a39896732;
        p1[1] = 0x5f8eb5a93e8fea99b62b9f88d8556c80dd726a4b30e84a36eeabaf3592937f27;
        p1[2] = 0x00000000000000000000000000000000086b990f3da2aeac0a36143b7d7c8244;
        p1[3] = 0x28215140db1bb859338764cb58458f081d92664f9053b50b3fbd2e4723121b68;
        p1[4] = 0x000000000000000000000000000000000f9e7ba9a86a8f7624aa2b42dcc8772e;
        p1[5] = 0x1af4ae115685e60abc2c9b90242167acef3d0be4050bf935eed7c3b6fc7ba77e;
        p1[6] = 0x000000000000000000000000000000000d22c3652d0dc6f0fc9316e14268477c;
        p1[7] = 0x2049ef772e852108d269d9c38dba1d4802e8dae479818184c08f9a569d878451;

        bytes32[8] memory p2;
        p2[0] = 0x00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051;
        p2[1] = 0xc6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8;
        p2[2] = 0x0000000000000000000000000000000013e02b6052719f607dacd3a088274f65;
        p2[3] = 0x596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e;
        p2[4] = 0x000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351a;
        p2[5] = 0xadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801;
        p2[6] = 0x000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99;
        p2[7] = 0xcb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be;

        bytes32 scalar = 0xe22466e928fdccef0de49e3503d2657d00494a00e764fd437bdafa05f5922b1f;

        for (uint256 i = 0; i < num_iterations; i++) {
            (bool ok, bytes memory result) = address(14).staticcall(abi.encode(p1, scalar, p2, scalar));
            require(ok, "G2MSM failed");
            p1 = abi.decode(result, (bytes32[8]));
        }
    }

    function run_bls_pairing(uint256 num_iterations) private {
        bytes32[4] memory p1;
        p1[0] = 0x0000000000000000000000000000000017f1d3a73197d7942695638c4fa9ac0f;
        p1[1] = 0xc3688c4f9774b905a14e3a3f171bac586c55e83ff97a1aeffb3af00adb22c6bb;
        p1[2] = 0x0000000000000000000000000000000008b3f481e3aaa0f1a09e30ed741d8ae4;
        p1[3] = 0xfcf5e095d5d00af600db18cb2c04b3edd03cc744a2888ae40caa232946c5e7e1;
        
        bytes32[8] memory p2;
        p2[0] = 0x00000000000000000000000000000000024aa2b2f08f0a91260805272dc51051;
        p2[1] = 0xc6e47ad4fa403b02b4510b647ae3d1770bac0326a805bbefd48056c8c121bdb8;
        p2[2] = 0x0000000000000000000000000000000013e02b6052719f607dacd3a088274f65;
        p2[3] = 0x596bd0d09920b61ab5da61bbdc7f5049334cf11213945d57e5ac7d055d042b7e;
        p2[4] = 0x000000000000000000000000000000000ce5d527727d6e118cc9cdc6da2e351a;
        p2[5] = 0xadfd9baa8cbdd3a76d429a695160d12c923ac9cc3baca289e193548608b82801;
        p2[6] = 0x000000000000000000000000000000000606c4a02ea734cc32acd2b02bc28b99;
        p2[7] = 0xcb3e287e85a763af267492ab572e99ab3f370d275cec1da1aaa9075ff05f79be;

        for (uint256 i = 0; i < num_iterations; i++) {
            (bool ok, bytes memory result) = address(15).staticcall(abi.encode(p1, p2));
            require(ok, "BLS Pairing failed");
        }
    }

    function run_map_g1(uint256 num_iterations) private {
        bytes32[2] memory p1;
        p1[0] = 0x0000000000000000000000000000000004090815ad598a06897dd89bcda860f2;
        p1[1] = 0x5837d54e897298ce31e6947378134d3761dc59a572154963e8c954919ecfa82d;

        for (uint256 i = 0; i < num_iterations; i++) {
            (bool ok, bytes memory result) = address(16).staticcall(abi.encode(p1));
            require(ok, "Map G1 failed");
            p1 = abi.decode(result, (bytes32[2]));
        }
    }

    function run_map_g2(uint256 num_iterations) private {
        bytes32[4] memory p1;
        p1[0] = 0x0000000000000000000000000000000018c16fe362b7dbdfa102e42bdfd3e2f4;
        p1[1] = 0xe6191d479437a59db4eb716986bf08ee1f42634db66bde97d6c16bbfd342b3b8;
        p1[2] = 0x000000000000000000000000000000000e37812ce1b146d998d5f92bdd5ada2a;
        p1[3] = 0x31bfd63dfe18311aa91637b5f279dd045763166aa1615e46a50d8d8f475f184e;

        for (uint256 i = 0; i < num_iterations; i++) {
            (bool ok, bytes memory result) = address(17).staticcall(abi.encode(p1));
            require(ok, "Map G2 failed");
            p1 = abi.decode(result, (bytes32[4]));
        }
    }
}
