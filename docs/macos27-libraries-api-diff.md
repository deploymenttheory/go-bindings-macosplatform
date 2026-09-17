# Metadata diff: SDK 26.5 → 27.0

## Summary

- Frameworks: 0 added, 0 removed, 11 changed
- Classes: 0 added, 0 removed
- C functions: 9 added, 0 removed, 3 signature change(s)

## AppleArchive

### Enum members added (1)

- `AACompressionAlgorithms.AA_COMPRESSION_ALGORITHM_LZRAVEN`

## Compression

### Enum members added (4)

- `_anon_COMPRESSION_LZ4.COMPRESSION_LZMESH`
- `_anon_COMPRESSION_LZ4.COMPRESSION_LZRAVEN`
- `compression_algorithm.COMPRESSION_LZMESH`
- `compression_algorithm.COMPRESSION_LZRAVEN`

## EndpointSecurity

### Enums added (19)

- `_anon_ES_BOOTSTRAP_TARGET_TYPE_PROCESS`
- `_anon_ES_DEADLINE_MISS_MODE_KILL`
- `dispatch_autorelease_frequency_t`
- `dispatch_block_flags_t`
- `es_bootstrap_target_type_t`
- `es_deadline_miss_mode_t`
- `filesec_property_t`
- `idtype_t`
- `ipc_info_object_type_t`
- `launch_data_type_t`
- `mach_vm_range_flags_t`
- `mach_vm_range_flavor_t`
- `mach_vm_range_tag_t`
- `os_clockid_t`
- `qos_class_t`
- `task_shared_region_stubs_t`
- `virtual_memory_guard_exception_code_t`
- `xpc_listener_create_flags_t`
- `xpc_session_create_flags_t`

### Enum members added (10)

- `_anon_ES_EVENT_TYPE_AUTH_EXEC.ES_EVENT_TYPE_AUTH_BOOTSTRAP_CHECK_IN`
- `_anon_ES_EVENT_TYPE_AUTH_EXEC.ES_EVENT_TYPE_AUTH_BOOTSTRAP_LOOK_UP`
- `_anon_ES_EVENT_TYPE_AUTH_EXEC.ES_EVENT_TYPE_AUTH_XPC_CONNECT`
- `_anon_ES_EVENT_TYPE_AUTH_EXEC.ES_EVENT_TYPE_NOTIFY_BOOTSTRAP_CHECK_IN`
- `_anon_ES_EVENT_TYPE_AUTH_EXEC.ES_EVENT_TYPE_NOTIFY_BOOTSTRAP_LOOK_UP`
- `es_event_type_t.ES_EVENT_TYPE_AUTH_BOOTSTRAP_CHECK_IN`
- `es_event_type_t.ES_EVENT_TYPE_AUTH_BOOTSTRAP_LOOK_UP`
- `es_event_type_t.ES_EVENT_TYPE_AUTH_XPC_CONNECT`
- `es_event_type_t.ES_EVENT_TYPE_NOTIFY_BOOTSTRAP_CHECK_IN`
- `es_event_type_t.ES_EVENT_TYPE_NOTIFY_BOOTSTRAP_LOOK_UP`

### Structs added (3)

- `es_event_bootstrap_check_in_t`
- `es_event_bootstrap_look_up_t`
- `es_lightweight_code_requirement_t`

### Functions added (9)

- `es_exec_entitlements`
- `es_get_deadline_max_milliseconds`
- `es_get_deadline_min_milliseconds`
- `es_get_deadline_miss_mode`
- `es_new_descendants_client`
- `es_set_deadline_max_milliseconds`
- `es_set_deadline_min_milliseconds`
- `es_set_deadline_miss_mode`
- `es_sync_client`

## Sandbox

### Externs removed (5)

- `kSBXProfileNoInternet`
- `kSBXProfileNoNetwork`
- `kSBXProfileNoWrite`
- `kSBXProfileNoWriteExceptTemporary`
- `kSBXProfilePureComputation`

## bsm

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## machhost

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## machinit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## machtime

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## machvm

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## oslog

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## xpc

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Function signature changes (3)

- `xpc_connection_send_message_with_reply`
  - old: `(xpc_connection_t _Nonnull, xpc_object_t _Nonnull, dispatch_queue_t _Nullable, XPC_SWIFT_SENDABLE xpc_handler_t _Nonnull) → `
  - new: `(xpc_connection_t _Nonnull, xpc_object_t _Nonnull, dispatch_queue_t _Nullable, xpc_handler_t _Nonnull __attribute__((swift_attr("@Sendable")))) → `
- `xpc_connection_set_event_handler`
  - old: `(xpc_connection_t _Nonnull, XPC_SWIFT_SENDABLE xpc_handler_t _Nonnull) → `
  - new: `(xpc_connection_t _Nonnull, xpc_handler_t _Nonnull __attribute__((swift_attr("@Sendable")))) → `
- `xpc_set_event_stream_handler`
  - old: `(const char * _Nonnull, dispatch_queue_t _Nullable, XPC_SWIFT_SENDABLE xpc_handler_t _Nonnull) → `
  - new: `(const char * _Nonnull, dispatch_queue_t _Nullable, xpc_handler_t _Nonnull __attribute__((swift_attr("@Sendable")))) → `
