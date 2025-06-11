pragma solidity ^0.8.13;

struct PrecompileConfig {
    address precompile_address;
    uint256 num_calls;
}

struct SimulatorConfig {
    uint256 load_accounts;
    uint256 update_accounts;
    uint256 delete_accounts;
    uint256 create_accounts;
    uint256 load_storage;
    uint256 update_storage;
    uint256 delete_storage;
    uint256 create_storage;
    PrecompileConfig[] precompiles;
}

contract Simulator {
    uint256 storage_slots_to_initialize;
    uint160 addresses_to_initialize;
    uint256 storage_chunk_size;
    uint160 address_chunk_size;

    mapping(uint256 => uint256) storage_slots;
    uint256 num_storage_initialized = 0;
    uint256 num_address_initialized = 0;
    uint256 num_storage_deleted = 0;

    // first storage slot with a value
    uint256 current_storage_slot_index = 0;

    constructor(
        uint256 _storage_slots_to_initialize,
        uint160 _addresses_to_initialize,
        uint256 _storage_chunk_size,
        uint160 _addresses_chunk_size
    ) public payable {
        storage_slots_to_initialize = _storage_slots_to_initialize;
        addresses_to_initialize = _addresses_to_initialize;
        storage_chunk_size = _storage_chunk_size;
        address_chunk_size = _addresses_chunk_size;
    }

    function num_storage_chunks() public view returns (uint256) {
        return storage_slots_to_initialize / storage_chunk_size;
    }

    function num_address_chunks() public view returns (uint256) {
        return addresses_to_initialize / address_chunk_size;
    }

    function fully_initialized() public view returns (bool) {
        return
            num_storage_initialized == storage_slots_to_initialize &&
            num_address_initialized == addresses_to_initialize;
    }

    function initialize_storage_chunk(uint256 chunk_index) public {
        uint256 start_index = chunk_index * storage_chunk_size;
        uint256 end_index = start_index + storage_chunk_size;

        for (uint256 i = start_index; i < end_index; i++) {
            storage_slots[i] = i;
        }
        num_storage_initialized += storage_chunk_size;
    }

    function initialize_address_chunk(uint256 chunk_index) public {
        uint160 start_index = uint160(chunk_index) * address_chunk_size;
        uint160 end_index = start_index + address_chunk_size;

        for (uint160 i = start_index; i < end_index; i++) {
            payable(address(i)).transfer(1);
        }
        num_address_initialized += address_chunk_size;
    }

    function run(SimulatorConfig calldata config) public {
        require(
            current_storage_slot_index +
                config.load_storage +
                config.update_storage <=
                num_storage_initialized,
            "Not enough storage slots to load/update"
        );

        // load storage slots using SLOAD in a loop. Ensure we're loading a unique storage slot each time.
        for (
            uint256 i = current_storage_slot_index;
            i < current_storage_slot_index + config.load_storage;
            i++
        ) {
            assembly {
                pop(sload(i))
            }
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
        // TODO: implement
    }
}
