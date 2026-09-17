# Metadata diff: SDK 26.5 → 27.0

## Summary

- Frameworks: 8 added, 0 removed, 220 changed
- Classes: 269 added, 7 removed
- Methods: 651 added, 28 removed, 279 signature change(s)

### Frameworks added (8)

- `AccessoryAccess`
- `AppIntentsTypeSupport`
- `ComputeGraph`
- `LinkSecurity`
- `MPSFunctions`
- `SpatialPreview`
- `StateReporting`
- `_ScreenCaptureKit_SwiftUI`

## AE

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ARKit

### Protocols added (15)

- `GCDevice`
- `OS_ar_accessories`
- `OS_ar_accessory`
- `OS_ar_accessory_anchor`
- `OS_ar_accessory_anchors`
- `OS_ar_accessory_tracking_configuration`
- `OS_ar_accessory_tracking_provider`
- `OS_ar_camera_region_anchor`
- `OS_ar_camera_region_anchors`
- `OS_ar_camera_region_configuration`
- `OS_ar_camera_region_provider`
- `OS_ar_coordinate_space_data`
- `OS_ar_reference_object_configuration`
- `OS_ar_shared_coordinate_space_configuration`
- `OS_ar_shared_coordinate_space_provider`

### Enums added (17)

- `_anon_ar_accessory_anchor_tracking_state_untracked`
- `_anon_ar_accessory_chirality_unspecified`
- `_anon_ar_accessory_source_type_device`
- `_anon_ar_accessory_tracking_error_code_accessory_loading_failed`
- `_anon_ar_camera_region_error_code_add_anchor_failed`
- `_anon_ar_camera_region_stabilization`
- `_anon_ar_transform_correction_none`
- `_anon_ar_world_anchor_sharing_availability_available`
- `ar_accessory_anchor_tracking_state_t`
- `ar_accessory_chirality_t`
- `ar_accessory_source_type_t`
- `ar_accessory_tracking_error_code_t`
- `ar_camera_region_camera_enhancement_t`
- `ar_camera_region_error_code_t`
- `ar_transform_correction_t`
- `ar_world_anchor_sharing_availability_t`
- `task_shared_region_stubs_t`

### Enum members added (7)

- `_anon_ar_authorization_type_none.ar_authorization_type_accessory_tracking`
- `ar_authorization_type_t.ar_authorization_type_accessory_tracking`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Functions added (111)

- `ar_accessories_add_accessories`
- `ar_accessories_add_accessory`
- `ar_accessories_create`
- `ar_accessories_enumerate_accessories`
- `ar_accessories_enumerate_accessories_f`
- `ar_accessories_get_count`
- `ar_accessories_remove_accessories`
- `ar_accessories_remove_accessory`
- `ar_accessory_anchor_create`
- `ar_accessory_anchor_get_accessory`
- `ar_accessory_anchor_get_anchor_from_location_transform_with_correction`
- `ar_accessory_anchor_get_angular_velocity`
- `ar_accessory_anchor_get_held_chirality`
- `ar_accessory_anchor_get_identifier`
- `ar_accessory_anchor_get_origin_from_anchor_transform`
- `ar_accessory_anchor_get_origin_from_anchor_transform_with_correction`
- `ar_accessory_anchor_get_timestamp`
- `ar_accessory_anchor_get_tracking_state`
- `ar_accessory_anchor_get_velocity`
- `ar_accessory_anchor_is_equal_to_accessory_anchor`
- `ar_accessory_anchor_is_held`
- `ar_accessory_anchor_is_tracked`
- `ar_accessory_anchors_enumerate_anchors`
- `ar_accessory_anchors_enumerate_anchors_f`
- `ar_accessory_anchors_get_count`
- `ar_accessory_copy_location_names`
- `ar_accessory_get_identifier`
- `ar_accessory_get_inherent_chirality`
- `ar_accessory_get_name`
- `ar_accessory_get_source_device`
- `ar_accessory_get_source_type`
- `ar_accessory_get_usdz_file_path`
- `ar_accessory_is_equal_to_accessory`
- `ar_accessory_load_from_device`
- `ar_accessory_load_from_device_f`
- `ar_accessory_tracking_configuration_create`
- `ar_accessory_tracking_configuration_set_accessories`
- `ar_accessory_tracking_provider_create`
- `ar_accessory_tracking_provider_get_latest_anchors`
- `ar_accessory_tracking_provider_get_required_authorization_type`
- `ar_accessory_tracking_provider_is_supported`
- `ar_accessory_tracking_provider_predict_anchor_at_timestamp`
- `ar_accessory_tracking_provider_set_update_handler`
- `ar_accessory_tracking_provider_set_update_handler_f`
- `ar_accessory_tracking_provider_update_accessories`
- `ar_accessory_tracking_provider_update_accessories_f`
- `ar_barcode_anchor_get_origin_from_anchor_transform_with_correction`
- `ar_camera_region_anchor_create_with_parameters`
- `ar_camera_region_anchor_get_camera_enhancement`
- `ar_camera_region_anchor_get_height`
- `ar_camera_region_anchor_get_identifier`
- `ar_camera_region_anchor_get_origin_from_anchor_transform`
- `ar_camera_region_anchor_get_origin_from_anchor_transform_with_correction`
- `ar_camera_region_anchor_get_pixel_buffer`
- `ar_camera_region_anchor_get_timestamp`
- `ar_camera_region_anchor_get_width`
- `ar_camera_region_anchor_is_equal_to_camera_region_anchor`
- `ar_camera_region_anchors_enumerate_anchors`
- `ar_camera_region_anchors_enumerate_anchors_f`
- `ar_camera_region_anchors_get_count`
- `ar_camera_region_configuration_create`
- `ar_camera_region_provider_add_camera_region_anchor`
- `ar_camera_region_provider_add_camera_region_anchor_f`
- `ar_camera_region_provider_create`
- `ar_camera_region_provider_get_required_authorization_type`
- `ar_camera_region_provider_is_supported`
- `ar_camera_region_provider_remove_camera_region_anchor`
- `ar_camera_region_provider_remove_camera_region_anchor_f`
- `ar_camera_region_provider_remove_camera_region_anchor_with_identifier`
- `ar_camera_region_provider_remove_camera_region_anchor_with_identifier_f`
- `ar_camera_region_provider_set_update_handler_for_anchor_with_identifier`
- `ar_camera_region_provider_set_update_handler_for_anchor_with_identifier_f`
- `ar_coordinate_space_data_copy_cfdata`
- `ar_coordinate_space_data_copy_recipient_identifers`
- `ar_coordinate_space_data_copy_recipient_identifiers`
- `ar_coordinate_space_data_create_from_cfdata`
- `ar_device_anchor_get_origin_from_anchor_transform_with_correction`
- `ar_environment_probe_anchor_get_origin_from_anchor_transform_with_correction`
- `ar_hand_anchor_get_origin_from_anchor_transform_with_correction`
- `ar_image_anchor_get_origin_from_anchor_transform_with_correction`
- `ar_mesh_anchor_get_origin_from_anchor_transform_with_correction`
- `ar_object_anchor_get_origin_from_anchor_transform_with_correction`
- `ar_plane_anchor_get_origin_from_anchor_transform_with_correction`
- `ar_reference_object_configuration_create`
- `ar_reference_object_configuration_enable_high_frame_rate_tracking`
- `ar_reference_object_configuration_is_equal_to_reference_object_configuration`
- `ar_reference_object_configuration_is_high_frame_rate_tracking_enabled`
- `ar_reference_object_load_from_url_with_configuration`
- `ar_reference_object_load_from_url_with_configuration_f`
- `ar_reference_object_load_with_name_and_configuration`
- `ar_reference_object_load_with_name_and_configuration_f`
- `ar_room_anchor_get_origin_from_anchor_transform_with_correction`
- `ar_shared_coordinate_provider_set_connected_participants_update_handler_f`
- `ar_shared_coordinate_space_configuration_create`
- `ar_shared_coordinate_space_provider_copy_next_coordinate_space_data`
- `ar_shared_coordinate_space_provider_create`
- `ar_shared_coordinate_space_provider_get_participant_identifier`
- `ar_shared_coordinate_space_provider_get_required_authorization_type`
- `ar_shared_coordinate_space_provider_is_sharing_enabled`
- `ar_shared_coordinate_space_provider_is_supported`
- `ar_shared_coordinate_space_provider_push_data`
- `ar_shared_coordinate_space_provider_set_connected_participants_update_handler`
- `ar_shared_coordinate_space_provider_set_sharing_status_update_handler`
- `ar_shared_coordinate_space_provider_set_sharing_status_update_handler_f`
- `ar_skeleton_joint_get_anchor_from_joint_transform_with_correction`
- `ar_skeleton_joint_get_parent_from_joint_transform_with_correction`
- `ar_world_anchor_get_origin_from_anchor_transform_with_correction`
- `ar_world_anchor_is_shared_with_nearby_participants`
- `ar_world_anchor_shared_with_nearby_participants_create`
- `ar_world_tracking_provider_set_world_anchor_sharing_availability_update_handler`
- `ar_world_tracking_provider_set_world_anchor_sharing_availability_update_handler_f`

### Function signature changes (42)

- `ar_barcode_anchor_copy_payload_data`
  - old: `(ar_barcode_anchor_t _Nonnull) → AR_OBJECT_RETURNS_RETAINED ar_data_t  _Nonnull`
  - new: `(ar_barcode_anchor_t _Nonnull) → ar_data_t  _Nonnull`
- `ar_barcode_detection_configuration_create`
  - old: `() → AR_OBJECT_RETURNS_RETAINED ar_barcode_detection_configuration_t  _Nonnull`
  - new: `() → ar_barcode_detection_configuration_t  _Nonnull`
- `ar_barcode_detection_provider_create`
  - old: `(ar_barcode_detection_configuration_t _Nonnull) → AR_OBJECT_RETURNS_RETAINED ar_barcode_detection_provider_t  _Nonnull`
  - new: `(ar_barcode_detection_configuration_t _Nonnull) → ar_barcode_detection_provider_t  _Nonnull`
- `ar_camera_frame_provider_create`
  - old: `() → AR_OBJECT_RETURNS_RETAINED ar_camera_frame_provider_t  _Nonnull`
  - new: `() → ar_camera_frame_provider_t  _Nonnull`
- `ar_camera_video_format_copy_supported_video_formats`
  - old: `(ar_camera_type_t, ar_camera_position_t) → AR_OBJECT_RETURNS_RETAINED ar_camera_video_formats_t  _Nonnull`
  - new: `(ar_camera_type_t, ar_camera_position_t) → ar_camera_video_formats_t  _Nonnull`
- `ar_data_providers_create`
  - old: `() → AR_OBJECT_RETURNS_RETAINED ar_data_providers_t  _Nonnull`
  - new: `() → ar_data_providers_t  _Nonnull`
- `ar_data_providers_create_with_data_providers`
  - old: `(ar_data_provider_t _Nonnull) → AR_OBJECT_RETURNS_RETAINED ar_data_providers_t  _Nonnull`
  - new: `(ar_data_provider_t _Nonnull) → ar_data_providers_t  _Nonnull`
- `ar_device_anchor_create`
  - old: `() → AR_OBJECT_RETURNS_RETAINED ar_device_anchor_t  _Nonnull`
  - new: `() → ar_device_anchor_t  _Nonnull`
- `ar_environment_light_estimation_configuration_create`
  - old: `() → AR_OBJECT_RETURNS_RETAINED ar_environment_light_estimation_configuration_t  _Nonnull`
  - new: `() → ar_environment_light_estimation_configuration_t  _Nonnull`
- `ar_environment_light_estimation_provider_create`
  - old: `(ar_environment_light_estimation_configuration_t _Nonnull) → AR_OBJECT_RETURNS_RETAINED ar_environment_light_estimation_provider_t  _Nonnull`
  - new: `(ar_environment_light_estimation_configuration_t _Nonnull) → ar_environment_light_estimation_provider_t  _Nonnull`
- `ar_hand_anchor_create`
  - old: `() → AR_OBJECT_RETURNS_RETAINED ar_hand_anchor_t  _Nonnull`
  - new: `() → ar_hand_anchor_t  _Nonnull`
- `ar_hand_skeleton_create`
  - old: `() → AR_OBJECT_RETURNS_RETAINED ar_hand_skeleton_t  _Nonnull`
  - new: `() → ar_hand_skeleton_t  _Nonnull`
- `ar_hand_tracking_configuration_create`
  - old: `() → AR_OBJECT_RETURNS_RETAINED ar_hand_tracking_configuration_t  _Nonnull`
  - new: `() → ar_hand_tracking_configuration_t  _Nonnull`
- `ar_hand_tracking_provider_create`
  - old: `(ar_hand_tracking_configuration_t _Nonnull) → AR_OBJECT_RETURNS_RETAINED ar_hand_tracking_provider_t  _Nonnull`
  - new: `(ar_hand_tracking_configuration_t _Nonnull) → ar_hand_tracking_provider_t  _Nonnull`
- `ar_image_tracking_configuration_create`
  - old: `() → AR_OBJECT_RETURNS_RETAINED ar_image_tracking_configuration_t  _Nonnull`
  - new: `() → ar_image_tracking_configuration_t  _Nonnull`
- `ar_image_tracking_provider_copy_all_image_anchors`
  - old: `(ar_image_tracking_provider_t _Nonnull) → AR_OBJECT_RETURNS_RETAINED ar_image_anchors_t  _Nonnull`
  - new: `(ar_image_tracking_provider_t _Nonnull) → ar_image_anchors_t  _Nonnull`
- `ar_image_tracking_provider_create`
  - old: `(ar_image_tracking_configuration_t _Nonnull) → AR_OBJECT_RETURNS_RETAINED ar_image_tracking_provider_t  _Nonnull`
  - new: `(ar_image_tracking_configuration_t _Nonnull) → ar_image_tracking_provider_t  _Nonnull`
- `ar_object_tracking_configuration_create`
  - old: `() → AR_OBJECT_RETURNS_RETAINED ar_object_tracking_configuration_t  _Nonnull`
  - new: `() → ar_object_tracking_configuration_t  _Nonnull`
- `ar_object_tracking_provider_copy_all_object_anchors`
  - old: `(ar_object_tracking_provider_t _Nonnull) → AR_OBJECT_RETURNS_RETAINED ar_object_anchors_t  _Nonnull`
  - new: `(ar_object_tracking_provider_t _Nonnull) → ar_object_anchors_t  _Nonnull`
- `ar_object_tracking_provider_create`
  - old: `(ar_object_tracking_configuration_t _Nonnull) → AR_OBJECT_RETURNS_RETAINED ar_object_tracking_provider_t  _Nonnull`
  - new: `(ar_object_tracking_configuration_t _Nonnull) → ar_object_tracking_provider_t  _Nonnull`
- `ar_plane_detection_configuration_create`
  - old: `() → AR_OBJECT_RETURNS_RETAINED ar_plane_detection_configuration_t  _Nonnull`
  - new: `() → ar_plane_detection_configuration_t  _Nonnull`
- `ar_plane_detection_provider_copy_all_plane_anchors`
  - old: `(ar_plane_detection_provider_t _Nonnull) → AR_OBJECT_RETURNS_RETAINED ar_plane_anchors_t  _Nonnull`
  - new: `(ar_plane_detection_provider_t _Nonnull) → ar_plane_anchors_t  _Nonnull`
- `ar_plane_detection_provider_create`
  - old: `(ar_plane_detection_configuration_t _Nonnull) → AR_OBJECT_RETURNS_RETAINED ar_plane_detection_provider_t  _Nonnull`
  - new: `(ar_plane_detection_configuration_t _Nonnull) → ar_plane_detection_provider_t  _Nonnull`
- `ar_reference_image_create_from_cgimage`
  - old: `(CGImageRef _Nonnull, CGImagePropertyOrientation, float) → AR_OBJECT_RETURNS_RETAINED ar_reference_image_t  _Nonnull`
  - new: `(CGImageRef _Nonnull, CGImagePropertyOrientation, float) → ar_reference_image_t  _Nonnull`
- `ar_reference_image_create_from_pixel_buffer`
  - old: `(CVPixelBufferRef _Nonnull, CGImagePropertyOrientation, float) → AR_OBJECT_RETURNS_RETAINED ar_reference_image_t  _Nonnull`
  - new: `(CVPixelBufferRef _Nonnull, CGImagePropertyOrientation, float) → ar_reference_image_t  _Nonnull`
- `ar_reference_images_create`
  - old: `() → AR_OBJECT_RETURNS_RETAINED ar_reference_images_t  _Nonnull`
  - new: `() → ar_reference_images_t  _Nonnull`
- `ar_reference_images_load_reference_images_in_group`
  - old: `(const char * _Nonnull, CFBundleRef _Nullable) → AR_OBJECT_RETURNS_RETAINED ar_reference_images_t  _Nonnull`
  - new: `(const char * _Nonnull, CFBundleRef _Nullable) → ar_reference_images_t  _Nonnull`
- `ar_reference_objects_create`
  - old: `() → AR_OBJECT_RETURNS_RETAINED ar_reference_objects_t  _Nonnull`
  - new: `() → ar_reference_objects_t  _Nonnull`
- `ar_room_tracking_configuration_create`
  - old: `() → AR_OBJECT_RETURNS_RETAINED ar_room_tracking_configuration_t  _Nonnull`
  - new: `() → ar_room_tracking_configuration_t  _Nonnull`
- `ar_room_tracking_provider_copy_all_room_anchors`
  - old: `(ar_room_tracking_provider_t _Nonnull) → AR_OBJECT_RETURNS_RETAINED ar_room_anchors_t  _Nonnull`
  - new: `(ar_room_tracking_provider_t _Nonnull) → ar_room_anchors_t  _Nonnull`
- `ar_room_tracking_provider_copy_current_room_anchor`
  - old: `(ar_room_tracking_provider_t _Nonnull) → AR_OBJECT_RETURNS_RETAINED ar_room_anchor_t  _Nullable`
  - new: `(ar_room_tracking_provider_t _Nonnull) → ar_room_anchor_t  _Nullable`
- `ar_room_tracking_provider_create`
  - old: `(ar_room_tracking_configuration_t _Nonnull) → AR_OBJECT_RETURNS_RETAINED ar_room_tracking_provider_t  _Nonnull`
  - new: `(ar_room_tracking_configuration_t _Nonnull) → ar_room_tracking_provider_t  _Nonnull`
- `ar_scene_reconstruction_configuration_create`
  - old: `() → AR_OBJECT_RETURNS_RETAINED ar_scene_reconstruction_configuration_t  _Nonnull`
  - new: `() → ar_scene_reconstruction_configuration_t  _Nonnull`
- `ar_scene_reconstruction_provider_copy_all_mesh_anchors`
  - old: `(ar_scene_reconstruction_provider_t _Nonnull) → AR_OBJECT_RETURNS_RETAINED ar_mesh_anchors_t  _Nonnull`
  - new: `(ar_scene_reconstruction_provider_t _Nonnull) → ar_mesh_anchors_t  _Nonnull`
- `ar_scene_reconstruction_provider_create`
  - old: `(ar_scene_reconstruction_configuration_t _Nonnull) → AR_OBJECT_RETURNS_RETAINED ar_scene_reconstruction_provider_t  _Nonnull`
  - new: `(ar_scene_reconstruction_configuration_t _Nonnull) → ar_scene_reconstruction_provider_t  _Nonnull`
- `ar_session_copy_data_providers`
  - old: `(ar_session_t _Nonnull) → AR_OBJECT_RETURNS_RETAINED ar_data_providers_t  _Nonnull`
  - new: `(ar_session_t _Nonnull) → ar_data_providers_t  _Nonnull`
- `ar_session_create`
  - old: `() → AR_OBJECT_RETURNS_RETAINED ar_session_t  _Nonnull`
  - new: `() → ar_session_t  _Nonnull`
- `ar_stereo_properties_configuration_create`
  - old: `() → AR_OBJECT_RETURNS_RETAINED ar_stereo_properties_configuration_t  _Nonnull`
  - new: `() → ar_stereo_properties_configuration_t  _Nonnull`
- `ar_stereo_properties_provider_create`
  - old: `(ar_stereo_properties_configuration_t _Nonnull) → AR_OBJECT_RETURNS_RETAINED ar_stereo_properties_provider_t  _Nonnull`
  - new: `(ar_stereo_properties_configuration_t _Nonnull) → ar_stereo_properties_provider_t  _Nonnull`
- `ar_world_anchor_create_with_origin_from_anchor_transform`
  - old: `(simd_float4x4) → AR_OBJECT_RETURNS_RETAINED ar_world_anchor_t  _Nonnull`
  - new: `(simd_float4x4) → ar_world_anchor_t  _Nonnull`
- `ar_world_tracking_configuration_create`
  - old: `() → AR_OBJECT_RETURNS_RETAINED ar_world_tracking_configuration_t  _Nonnull`
  - new: `() → ar_world_tracking_configuration_t  _Nonnull`
- `ar_world_tracking_provider_create`
  - old: `(ar_world_tracking_configuration_t _Nonnull) → AR_OBJECT_RETURNS_RETAINED ar_world_tracking_provider_t  _Nonnull`
  - new: `(ar_world_tracking_configuration_t _Nonnull) → ar_world_tracking_provider_t  _Nonnull`

### Externs added (3)

- `ar_accessory_location_name_aim`
- `ar_accessory_location_name_grip`
- `ar_accessory_location_name_grip_surface`

## ATS

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## AVFAudio

### Classes added (3)

- `AVAudioSessionDeactivationContext`
- `AVAudioSessionInterruptionContext`
- `AVAudioSessionResumptionContext`

### Methods added (14)

- `AVAudioEngine -connect:to:format:error:`
- `AVAudioEngine -connect:to:fromBus:toBus:format:error:`
- `AVAudioEngine -connect:toConnectionPoints:fromBus:format:error:`
- `AVAudioEngine -connectMIDI:to:format:eventListProvider:`
- `AVAudioEngine -connectMIDI:toNodes:format:eventListProvider:`
- `AVAudioFormat -initWithFormatDescription:`
- `AVAudioInputNode -setRealtimeSafeManualRenderingInputPCMFormat:inputBlock:`
- `AVAudioNode -installTapOnBus:bufferSize:format:error:block:`
- `AVAudioPlayerNode -playAndReturnError:`
- `AVAudioPlayerNode -playAtTime:error:`
- `AVAudioSession -deactivateWithOptions:completionHandler:`
- `AVAudioSinkNode -initWithRealtimeSafeReceiverBlock:`
- `AVAudioSourceNode -initWithFormat:realtimeSafeRenderBlock:`
- `AVAudioSourceNode -initWithRealtimeSafeRenderBlock:`

### Method signature changes (6)

- `AVAudioEngine -musicSequence`
  - old: `() → MusicSequence _Nullable`
  - new: `() → MusicSequence`
- `AVAudioEngine -setMusicSequence:`
  - old: `(MusicSequence _Nullable) → void`
  - new: `(MusicSequence) → void`
- `AVAudioIONode -audioUnit`
  - old: `() → AudioUnit _Nullable`
  - new: `() → AudioUnit`
- `AVAudioNode -AUAudioUnit`
  - old: `() → AUAudioUnit * _Nonnull`
  - new: `() → AUAudioUnit *`
- `AVAudioUnit -AUAudioUnit`
  - old: `() → AUAudioUnit * _Nonnull`
  - new: `() → AUAudioUnit *`
- `AVAudioUnit -audioUnit`
  - old: `() → AudioUnit _Nonnull`
  - new: `() → AudioUnit`

### Enums added (4)

- `AVAudioSessionDeactivationOptions`
- `AVAudioSessionDeactivationSource`
- `AVAudioSessionResumptionRecommendation`
- `task_shared_region_stubs_t`

### Enum members added (6)

- `AVAudioUnitReverbPreset.AVAudioUnitReverbPresetOutdoorGeneral`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Externs added (6)

- `AVAudioSessionDeactivationContextKey`
- `AVAudioSessionDidBecomeActiveNotification`
- `AVAudioSessionDidBecomeInactiveNotification`
- `AVAudioSessionPortMediaDeviceExtension`
- `AVAudioSessionResumptionContextKey`
- `AVAudioSessionResumptionRecommendationNotification`

## AVFoundation

### Classes added (18)

- `AVAssetExportSessionResumptionState`
- `AVAssetTrackPlan`
- `AVAssetVideoTrackPlan`
- `AVAssetWritingPlanner`
- `AVAssetWritingPlannerProgress`
- `AVCaptureAncillaryDataEncoder`
- `AVCaptureBroadcastVideoOutput`
- `AVMetadataCinematicVideoMetadataObject`
- `AVMetadataFocusTrackedObject`
- `AVMetricPlaybackModeSwitchEvent`
- `AVPlannedSegmentConfiguration`
- `AVPlannedSegmentWritingRequest`
- `AVPlannedVideoSegmentConfiguration`
- `AVPlannedVideoSegmentWritingRequest`
- `AVPlayerItemSampleBufferOutput`
- `AVPlayerItemSampleBufferOutputAudioConfiguration`
- `AVPlayerItemSampleBufferOutputConfiguration`
- `AVProVideoStorage`

### Methods added (67)

- `AVAsset -constituentFileURLs`
- `AVAssetExportSession -configureForResumableExportWithCompletionHandler:`
- `AVAssetWriter -isProVideoStorageSupported`
- `AVAssetWriter -setUsesProVideoStorage:`
- `AVAssetWriter -usesProVideoStorage`
- `AVCaptureConnection -automaticallyEnablesLowLightVideoNoiseReduction`
- `AVCaptureConnection -isLowLightVideoNoiseReductionEnabled`
- `AVCaptureConnection -isLowLightVideoNoiseReductionSupported`
- `AVCaptureConnection -setAutomaticallyEnablesLowLightVideoNoiseReduction:`
- `AVCaptureConnection -setLowLightVideoNoiseReductionEnabled:`
- `AVCaptureDevice -activeExposureSignals`
- `AVCaptureDevice -autoExposureLensApertureRateLimit`
- `AVCaptureDevice -automaticallyAdjustsExposureDuration`
- `AVCaptureDevice -automaticallyAdjustsISO`
- `AVCaptureDevice -automaticallyAdjustsLensAperture`
- `AVCaptureDevice -automaticallyEnablesExposureSignals`
- `AVCaptureDevice -continuousAutoFocusTrackingLensPositionBias`
- `AVCaptureDevice -enabledExposureSignals`
- `AVCaptureDevice -isAdjustingSignalCompensationDelayWhileRunningSupported`
- `AVCaptureDevice -isContinuousAutoFocusTrackingEnabled`
- `AVCaptureDevice -isContinuousAutoFocusTrackingSubjectAcquired`
- `AVCaptureDevice -isPrimaryConstituentDeviceSwitchingBehaviorLockedWithDeviceSupported`
- `AVCaptureDevice -setAutoExposureLensApertureRateLimit:`
- `AVCaptureDevice -setAutomaticallyEnablesExposureSignals:`
- `AVCaptureDevice -setContinuousAutoFocusTrackingEnabled:`
- `AVCaptureDevice -setContinuousAutoFocusTrackingLensPositionBias:`
- `AVCaptureDevice -setEnabledExposureSignals:`
- `AVCaptureDevice -setExposureModeCustomWithLensAperture:duration:ISO:completionHandler:`
- `AVCaptureDevice -setPrimaryConstituentDeviceSwitchingBehaviorLockedWithDevice:`
- `AVCaptureDevice -supportedExposureSignals`
- `AVCaptureDeviceFormat -defaultLensAperture`
- `AVCaptureDeviceFormat -isCinematicVideoMetadataCaptureSupported`
- `AVCaptureDeviceFormat -isContinuousAutoFocusTrackingSupported`
- `AVCaptureDeviceFormat -isLowLightVideoNoiseReductionSupported`
- `AVCaptureDeviceFormat -maxLensAperture`
- `AVCaptureDeviceFormat -minLensAperture`
- `AVCaptureDeviceFormat -recommendedLensApertureStops`
- `AVCaptureDeviceFormat -supportsExposureModeCustomWithLensAperture:duration:ISO:`
- `AVCaptureDeviceRotationCoordinator -videoRotationAngleRelativeToDeviceOrientation:`
- `AVCaptureMovieFileOutput -automaticallyAdjustsCinematicVideoMetadataCaptureEnabled`
- `AVCaptureMovieFileOutput -isCinematicVideoMetadataCaptureEnabled`
- `AVCaptureMovieFileOutput -isCinematicVideoMetadataCaptureSupported`
- `AVCaptureMovieFileOutput -isProVideoStorageSupported`
- `AVCaptureMovieFileOutput -setAutomaticallyAdjustsCinematicVideoMetadataCaptureEnabled:`
- `AVCaptureMovieFileOutput -setCinematicVideoMetadataCaptureEnabled:`
- `AVCaptureMovieFileOutput -setUsesProVideoStorage:`
- `AVCaptureMovieFileOutput -usesProVideoStorage`
- `AVContentKeyRequest -makeOptionalStreamingContentKeyRequestDataForApp:contentIdentifier:options:completionHandler:`
- `AVExternalStorageDevice -reasonsNotRecommendedForCaptureUse`
- `AVExternalSyncDevice -isSignalCompensationDelaySupported`
- `AVPlayer -disconnectedFromSystemAudio`
- `AVPlayer -setDisconnectedFromSystemAudio:completionHandler:`
- `AVPlayerItem -fetchAccessLogWithCompletionHandler:`
- `AVPlayerItem -fetchErrorLogWithCompletionHandler:`
- `AVPlayerItem -selectableMediaSelectionOptionsInMediaSelectionGroup:`
- `AVSampleBufferAudioRenderer -enqueueSampleBuffer:`
- `AVSampleBufferAudioRenderer -flush`
- `AVSampleBufferAudioRenderer -hasSufficientMediaDataForReliablePlaybackStart`
- `AVSampleBufferAudioRenderer -isReadyForMoreMediaData`
- `AVSampleBufferAudioRenderer -requestMediaDataWhenReadyOnQueue:usingBlock:`
- `AVSampleBufferAudioRenderer -stopRequestingMediaData`
- `AVSampleBufferVideoRenderer -enqueueSampleBuffer:`
- `AVSampleBufferVideoRenderer -flush`
- `AVSampleBufferVideoRenderer -hasSufficientMediaDataForReliablePlaybackStart`
- `AVSampleBufferVideoRenderer -isReadyForMoreMediaData`
- `AVSampleBufferVideoRenderer -requestMediaDataWhenReadyOnQueue:usingBlock:`
- `AVSampleBufferVideoRenderer -stopRequestingMediaData`

### Method signature changes (1)

- `AVSampleBufferVideoRenderer -error`
  - old: `() → NSError * _Nullable`
  - new: `() → NSError *`

### Protocols added (2)

- `AVCaptureBroadcastVideoOutputDelegate`
- `AVPlayerItemSampleBufferOutputDelegate`

### Enums added (4)

- `AVAudioMixInputParametersTrackID`
- `AVCaptureBroadcastVideoOutputDroppedFrameReplacementPolicy`
- `AVMetricPlaybackMode`
- `task_shared_region_stubs_t`

### Enum members added (10)

- `AVCaptureSystemPressureFactors.AVCaptureSystemPressureFactorBatteryStress`
- `AVError.AVErrorExternalSyncDeviceFrequencyHigherThanSpecified`
- `AVError.AVErrorExternalSyncDeviceFrequencyLowerThanSpecified`
- `AVError.AVErrorFollowExternalSyncFailed`
- `AVError.AVErrorNotEnoughSpaceForProVideoStorageReplenishment`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Structs added (1)

- `AVPlannedVideoSegmentBoundaryGuidelines`

### Externs added (29)

- `AVAssetExportSessionResumptionFailureReasonIncompatibleSessionSettings`
- `AVAssetExportSessionResumptionFailureReasonIncompatibleTemporaryDirectoryContents`
- `AVAssetExportSessionResumptionFailureReasonTemporaryDirectoryDoesNotExist`
- `AVAssetExportSessionResumptionFailureReasonUnsupportedForPresetOnPlatform`
- `AVCaptureAncillaryDataUserKeyRDD18InstanceUID`
- `AVCaptureAncillaryDataUserKeyRDD18UDAMSetVersion`
- `AVCaptureAncillaryDataUserKeyRDD18UserItems`
- `AVCaptureDeviceExposureSignalDocument`
- `AVCaptureDeviceExposureSignalFlicker`
- `AVCaptureDeviceExposureSignalGroupPhoto`
- `AVCaptureDeviceExposureSignalStarburst`
- `AVCaptureDeviceExposureSignalSubjectMotion`
- `AVCaptureExposureDurationAuto`
- `AVCaptureISOAuto`
- `AVCaptureLensApertureAuto`
- `AVCaptureLensApertureCurrent`
- `AVExternalStorageDeviceReasonNotRecommendedForCaptureUseEncrypted`
- `AVExternalStorageDeviceReasonNotRecommendedForCaptureUseSlowWritingSpeed`
- `AVExternalStorageDeviceReasonNotRecommendedForCaptureUseUnknownWritingSpeed`
- `AVExternalStorageDeviceReasonNotRecommendedForCaptureUseUnsupportedFileSystem`
- `AVMetadataObjectTypeCinematicVideoMetadata`
- `AVMetadataObjectTypeFocusTrackedObject`
- `AVProVideoStorageBusyReasonAdjustingCapacity`
- `AVProVideoStorageBusyReasonCapturing`
- `AVProVideoStorageBusyReasonReplenishing`
- `AVSampleBufferVideoRendererRequiresFlushToResumeDecodingDidChangeNotificationRequiresFlushKey`
- `AVVideoLogTransferFunctionKey`
- `AVVideoLogTransferFunction_AppleLog`
- `AVVideoLogTransferFunction_AppleLog2`

### Deprecation changes (1)

- `AVAggregateAssetDownloadTask: deprecated API_TO_BE_DEPRECATED → 27.0`

## AVKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enums removed (1)

- `AVLegibleMediaOptionsMenuType`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## AVRouting

### Method signature changes (2)

- `AVRoutingPlaybackArbiter -preferredParticipantForNonMixableAudioRoutes`
  - old: `() → id<AVRoutingPlaybackParticipant>`
  - new: `() → id<AVRoutingPlaybackParticipant> _Nullable`
- `AVRoutingPlaybackArbiter -setPreferredParticipantForNonMixableAudioRoutes:`
  - old: `(id<AVRoutingPlaybackParticipant>) → void`
  - new: `(id<AVRoutingPlaybackParticipant> _Nullable) → void`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Enum base type changes (1)

- `nw_link_quality_t`
  - old: `int32`
  - new: `uint8`

## Accelerate

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## Accessibility

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Functions added (1)

- `AXApplicationAccessibilityEnabled`

### Externs added (2)

- `AXApplicationAccessibilityEnabledDidChangeNotification`
- `AXSpeechAttributeSSML`

## Accounts

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## AdServices

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## AdSupport

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## AddressBook

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## AppIntents

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## AppKit

### Classes added (6)

- `NSRefreshController`
- `NSStatusItemExpandedInterfaceSession`
- `NSTextSelectionManager`
- `NSViewCornerConfiguration`
- `NSViewCornerRadii`
- `NSViewCornerRadius`

### Methods added (60)

- `NSControl -addTarget:action:forControlEvents:`
- `NSControl -removeTarget:action:forControlEvents:`
- `NSEvent +isTouchSwipeNavigationEnabled`
- `NSGestureRecognizer -isCancellableByScrollGesture`
- `NSGestureRecognizer -setCancellableByScrollGesture:`
- `NSGlassEffectView -effectIsInteractive`
- `NSGlassEffectView -setEffectIsInteractive:`
- `NSMenuItem -preferredImageVisibility`
- `NSMenuItem -setPreferredImageVisibility:`
- `NSPanGestureRecognizer -maximumNumberOfTouches`
- `NSPanGestureRecognizer -minimumNumberOfTouches`
- `NSPanGestureRecognizer -setMaximumNumberOfTouches:`
- `NSPanGestureRecognizer -setMinimumNumberOfTouches:`
- `NSScreen -touchCapabilities`
- `NSScrollView -isTouchScrollingEnabled`
- `NSScrollView -maximumNumberOfTouchesForScrolling`
- `NSScrollView -minimumNumberOfTouchesForScrolling`
- `NSScrollView -refreshController`
- `NSScrollView -scrollGestureForRelationships`
- `NSScrollView -setMaximumNumberOfTouchesForScrolling:`
- `NSScrollView -setMinimumNumberOfTouchesForScrolling:`
- `NSScrollView -setRefreshController:`
- `NSScrollView -setTouchScrollingEnabled:`
- `NSSegmentedControl -role`
- `NSSegmentedControl -setRole:`
- `NSShadow -initWithCoder:`
- `NSSpellChecker -ignoreGrammarRange:inSentence:inSpellDocumentWithTag:`
- `NSStatusItem -expandedInterfaceDelegate`
- `NSStatusItem -expandedInterfaceSession`
- `NSStatusItem -setExpandedInterfaceDelegate:`
- `NSTextBlock -borderColorForRectEdge:`
- `NSTextBlock -initWithCoder:`
- `NSTextBlock -setBorderColor:rectEdge:`
- `NSTextBlock -setWidth:type:forLayer:rectEdge:`
- `NSTextBlock -widthForLayer:rectEdge:`
- `NSTextBlock -widthValueTypeForLayer:rectEdge:`
- `NSTextField -borderShape`
- `NSTextField -setBorderShape:`
- `NSTextTableBlock -initWithCoder:`
- `NSTextView -registerTextAttachmentViewProviderReusePolicy:forTextAttachmentViewProviderType:`
- `NSTextView -textViewportLayoutController:configureRenderingSurfaceForTextLayoutFragment:`
- `NSTextView -textViewportLayoutControllerDidLayout:`
- `NSTextView -textViewportLayoutControllerReceivedSetNeedsLayout:`
- `NSTextView -textViewportLayoutControllerWillLayout:`
- `NSTextView -viewportBoundsForTextViewportLayoutController:`
- `NSTextViewportLayoutController -renderingSurfaceForKey:`
- `NSToolbarItemGroup -role`
- `NSToolbarItemGroup -setRole:`
- `NSView -beginDraggingSessionWithItems:gesture:source:`
- `NSView -cornerConfiguration`
- `NSView -effectiveCornerRadii`
- `NSView -exclusiveGestureBehavior`
- `NSView -invalidateCornerConfiguration`
- `NSView -setExclusiveGestureBehavior:`
- `NSView -setTextSelectionManager:`
- `NSView -textSelectionManager`
- `NSView -viewDidChangeEffectiveCornerRadii`
- `NSWritingToolsCoordinator -cancelTextAnimationsWithIdentifiers:`
- `NSWritingToolsCoordinator -showGrammarPresentationForRange:inContext:`
- `NSWritingToolsCoordinator -startTextAnimation:forRange:inContext:writingDirection:`

### Method signature changes (8)

- `NSColor +alternateSelectedControlColor`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT NSColor *`
  - new: `() → NSColor *`
- `NSColor +controlAlternatingRowBackgroundColors`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT NSArray<NSColor *> *`
  - new: `() → NSArray<NSColor *> *`
- `NSColor +secondarySelectedControlColor`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT NSColor *`
  - new: `() → NSColor *`
- `NSShadow -setShadowOffset:`
  - old: `(NSSize) → void`
  - new: `(CGSize) → void`
- `NSShadow -shadowOffset`
  - old: `() → NSSize`
  - new: `() → CGSize`
- `NSTextBlock -drawBackgroundWithFrame:inView:characterRange:layoutManager:`
  - old: `(NSRect, NSView * _Nonnull, NSRange, NSLayoutManager * _Nonnull) → void`
  - new: `(NSRect, NSView * _Nullable, NSRange, NSLayoutManager * _Nonnull) → void`
- `NSTextTable -numberOfColumns`
  - old: `() → NSUInteger`
  - new: `() → NSInteger`
- `NSTextTable -setNumberOfColumns:`
  - old: `(NSUInteger) → void`
  - new: `(NSInteger) → void`

### Protocols added (4)

- `NSStatusItemExpandedInterfaceDelegate`
- `NSTextSelectionManagerDelegate`
- `NSTextViewportRenderingSurface`
- `NSTextViewportRenderingSurfaceKey`

### Enums added (10)

- `NSControlEvents`
- `NSMenuItemImageVisibility`
- `NSScreenTouchCapabilities`
- `NSSegmentedControlRole`
- `NSTextAttachmentViewProviderReusePolicy`
- `NSTextSelectionMode`
- `NSToolbarItemGroupRole`
- `NSViewExclusiveGestureBehavior`
- `NSWritingToolsCoordinatorTextDecoration`
- `task_shared_region_stubs_t`

### Enum members added (27)

- `NSApplicationPresentationOptions.NSApplicationPresentationDisableScreenCornerInteractions`
- `NSTextBlockDimension.NSTextBlockDimensionHeight`
- `NSTextBlockDimension.NSTextBlockDimensionMaximumHeight`
- `NSTextBlockDimension.NSTextBlockDimensionMaximumWidth`
- `NSTextBlockDimension.NSTextBlockDimensionMinimumHeight`
- `NSTextBlockDimension.NSTextBlockDimensionMinimumWidth`
- `NSTextBlockDimension.NSTextBlockDimensionWidth`
- `NSTextBlockLayer.NSTextBlockLayerBorder`
- `NSTextBlockLayer.NSTextBlockLayerMargin`
- `NSTextBlockLayer.NSTextBlockLayerPadding`
- `NSTextBlockValueType.NSTextBlockValueTypeAbsolute`
- `NSTextBlockValueType.NSTextBlockValueTypePercentage`
- `NSTextBlockVerticalAlignment.NSTextBlockVerticalAlignmentBaseline`
- `NSTextBlockVerticalAlignment.NSTextBlockVerticalAlignmentBottom`
- `NSTextBlockVerticalAlignment.NSTextBlockVerticalAlignmentMiddle`
- `NSTextBlockVerticalAlignment.NSTextBlockVerticalAlignmentTop`
- `NSTextTableLayoutAlgorithm.NSTextTableLayoutAlgorithmAutomatic`
- `NSTextTableLayoutAlgorithm.NSTextTableLayoutAlgorithmFixed`
- `NSWritingToolsCoordinatorTextAnimation.NSWritingToolsCoordinatorTextAnimationIndicateGrammar`
- `NSWritingToolsCoordinatorTextReplacementReason.NSWritingToolsCoordinatorTextReplacementReasonAccepted`
- `NSWritingToolsCoordinatorTextReplacementReason.NSWritingToolsCoordinatorTextReplacementReasonRejected`
- `NSWritingToolsCoordinatorTextReplacementReason.NSWritingToolsCoordinatorTextReplacementReasonTemporary`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Enum base type changes (4)

- `NSTextBlockDimension`
  - old: `uint64`
  - new: `int64`
- `NSTextBlockValueType`
  - old: `uint64`
  - new: `int64`
- `NSTextBlockVerticalAlignment`
  - old: `uint64`
  - new: `int64`
- `NSTextTableLayoutAlgorithm`
  - old: `uint64`
  - new: `int64`

### Externs added (3)

- `NSPaperMarginDocumentAttribute`
- `NSTextCheckingAutomaticCapitalizationEnabledKey`
- `NSTextCheckingWaitForAllGrammarCheckingResultsKey`

## AppTrackingTransparency

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## AppleScriptObjC

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ApplicationServices

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## AudioToolbox

### Classes added (1)

- `AUHeadTrackingBinauralRenderer`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (8)

- `AUReverbRoomType.kReverbRoomType_OutdoorGeneral`
- `_anon_kAudioUnitType_Output.kAudioUnitType_HeadTrackingBinauralRenderer`
- `_anon_kReverb2Param_DryWetMix.kReverb2Param_LegacyMode`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Function signature changes (1)

- `AudioComponentCopyIcon`
  - old: `(AudioComponent _Nonnull) → API_AVAILABLE NSImage * _Nullable`
  - new: `(AudioComponent _Nonnull) → NSImage * _Nullable`

## AudioUnit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## AudioVideoBridging

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## AuthenticationServices

### Methods added (8)

- `ASAuthorizationProviderExtensionLoginConfiguration -authorizationURL`
- `ASAuthorizationProviderExtensionLoginConfiguration -authorizationURLKeypath`
- `ASAuthorizationProviderExtensionLoginConfiguration -fallbackFederationType`
- `ASAuthorizationProviderExtensionLoginConfiguration -includePlatformSSOAuthorizationScopes`
- `ASAuthorizationProviderExtensionLoginConfiguration -setAuthorizationURL:`
- `ASAuthorizationProviderExtensionLoginConfiguration -setAuthorizationURLKeypath:`
- `ASAuthorizationProviderExtensionLoginConfiguration -setFallbackFederationType:`
- `ASAuthorizationProviderExtensionLoginConfiguration -setIncludePlatformSSOAuthorizationScopes:`

### Method signature changes (2)

- `ASAuthorizationProviderExtensionLoginManager -loginUserName`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT NSString *`
  - new: `() → NSString *`
- `ASAuthorizationProviderExtensionLoginManager -setLoginUserName:`
  - old: `(API_DEPRECATED_WITH_REPLACEMENT NSString *) → void`
  - new: `(NSString *) → void`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (10)

- `ASAuthorizationProviderExtensionAuthenticationMethod.ASAuthorizationProviderExtensionAuthenticationMethodOpenID`
- `ASAuthorizationProviderExtensionFederationType.ASAuthorizationProviderExtensionFederationTypeDynamicOpenID`
- `ASAuthorizationProviderExtensionFederationType.ASAuthorizationProviderExtensionFederationTypeOpenID`
- `ASAuthorizationProviderExtensionRequestOptions.ASAuthorizationProviderExtensionRequestOptionsOpenIDFallback`
- `ASAuthorizationProviderExtensionSupportedGrantTypes.ASAuthorizationProviderExtensionSupportedGrantTypesTokenExchange`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## AutomaticAssessmentConfiguration

### Classes added (2)

- `AEAssessmentBinaryExecutable`
- `AEAssessmentBinaryExecutableConfiguration`

### Methods added (62)

- `AEAssessmentConfiguration -allowedAppleMenuItems`
- `AEAssessmentConfiguration -allowedDirectoriesAndFiles`
- `AEAssessmentConfiguration -allowedMenuBarItems`
- `AEAssessmentConfiguration -allowsAccessibilityAlternativeInputMethods`
- `AEAssessmentConfiguration -allowsAccessibilityBackgroundSounds`
- `AEAssessmentConfiguration -allowsAccessibilityFullKeyboardAccess`
- `AEAssessmentConfiguration -allowsAccessibilityHoverText`
- `AEAssessmentConfiguration -allowsAccessibilityLiveSpeech`
- `AEAssessmentConfiguration -allowsAccessibilitySpokenContent`
- `AEAssessmentConfiguration -allowsAccessibilitySwitchControl`
- `AEAssessmentConfiguration -allowsAccessibilityVoiceControl`
- `AEAssessmentConfiguration -allowsAccessibilityVoiceOver`
- `AEAssessmentConfiguration -allowsAccessibilityZoom`
- `AEAssessmentConfiguration -allowsAutoFill`
- `AEAssessmentConfiguration -allowsDock`
- `AEAssessmentConfiguration -allowsForceQuitKeyboardShortcuts`
- `AEAssessmentConfiguration -allowsLockdownMode`
- `AEAssessmentConfiguration -allowsMenuBar`
- `AEAssessmentConfiguration -allowsOnlyParticipantsToRun`
- `AEAssessmentConfiguration -allowsPrivateRelay`
- `AEAssessmentConfiguration -allowsStructuralInput`
- `AEAssessmentConfiguration -allowsUserScriptExecution`
- `AEAssessmentConfiguration -allowsVirtualMachine`
- `AEAssessmentConfiguration -configurationsByBinaryExecutable`
- `AEAssessmentConfiguration -removeBinaryExecutable:`
- `AEAssessmentConfiguration -requiresManagedDevice`
- `AEAssessmentConfiguration -requiresReleaseOS`
- `AEAssessmentConfiguration -requiresSIP`
- `AEAssessmentConfiguration -requiresSingleUser`
- `AEAssessmentConfiguration -requiresUserAccountType`
- `AEAssessmentConfiguration -setAllowedAppleMenuItems:`
- `AEAssessmentConfiguration -setAllowedDirectoriesAndFiles:`
- `AEAssessmentConfiguration -setAllowedMenuBarItems:`
- `AEAssessmentConfiguration -setAllowsAccessibilityAlternativeInputMethods:`
- `AEAssessmentConfiguration -setAllowsAccessibilityBackgroundSounds:`
- `AEAssessmentConfiguration -setAllowsAccessibilityFullKeyboardAccess:`
- `AEAssessmentConfiguration -setAllowsAccessibilityHoverText:`
- `AEAssessmentConfiguration -setAllowsAccessibilityLiveSpeech:`
- `AEAssessmentConfiguration -setAllowsAccessibilitySpokenContent:`
- `AEAssessmentConfiguration -setAllowsAccessibilitySwitchControl:`
- `AEAssessmentConfiguration -setAllowsAccessibilityVoiceControl:`
- `AEAssessmentConfiguration -setAllowsAccessibilityVoiceOver:`
- `AEAssessmentConfiguration -setAllowsAccessibilityZoom:`
- `AEAssessmentConfiguration -setAllowsAutoFill:`
- `AEAssessmentConfiguration -setAllowsDock:`
- `AEAssessmentConfiguration -setAllowsForceQuitKeyboardShortcuts:`
- `AEAssessmentConfiguration -setAllowsLockdownMode:`
- `AEAssessmentConfiguration -setAllowsMenuBar:`
- `AEAssessmentConfiguration -setAllowsOnlyParticipantsToRun:`
- `AEAssessmentConfiguration -setAllowsPrivateRelay:`
- `AEAssessmentConfiguration -setAllowsStructuralInput:`
- `AEAssessmentConfiguration -setAllowsUserScriptExecution:`
- `AEAssessmentConfiguration -setAllowsVirtualMachine:`
- `AEAssessmentConfiguration -setConfiguration:forBinaryExecutable:`
- `AEAssessmentConfiguration -setRequiresManagedDevice:`
- `AEAssessmentConfiguration -setRequiresReleaseOS:`
- `AEAssessmentConfiguration -setRequiresSIP:`
- `AEAssessmentConfiguration -setRequiresSingleUser:`
- `AEAssessmentConfiguration -setRequiresUserAccountType:`
- `AEAssessmentParticipantConfiguration -allowedMenuItemLanguageIdentifiers`
- `AEAssessmentParticipantConfiguration -allowedMenuItemsForLanguageIdentifier:`
- `AEAssessmentParticipantConfiguration -setAllowedMenuItems:forLanguageIdentifier:`

### Method signature changes (1)

- `AEAssessmentApplication -teamIdentifier`
  - old: `() → NSString *`
  - new: `() → NSString * _Nullable`

### Enums added (2)

- `AEUserAccountType`
- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Externs added (19)

- `AEAppleMenuItemAboutThisMac`
- `AEAppleMenuItemAppStore`
- `AEAppleMenuItemForceQuit`
- `AEAppleMenuItemLocation`
- `AEAppleMenuItemLockScreen`
- `AEAppleMenuItemLogout`
- `AEAppleMenuItemRecent`
- `AEAppleMenuItemRestart`
- `AEAppleMenuItemShutDown`
- `AEAppleMenuItemSleep`
- `AEAppleMenuItemSystemInformation`
- `AEAppleMenuItemSystemSettings`
- `AEMenuBarItemBattery`
- `AEMenuBarItemBluetooth`
- `AEMenuBarItemClock`
- `AEMenuBarItemDisplays`
- `AEMenuBarItemKeyboard`
- `AEMenuBarItemVolume`
- `AEMenuBarItemWifi`

## Automator

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## BackgroundAssets

### Methods added (17)

- `BAAssetPack -language`
- `BAAssetPackManager -URLForPath:asLocalizedForLanguage:error:`
- `BAAssetPackManager -contentsAtPath:asLocalizedForLanguage:options:error:`
- `BAAssetPackManager -ensureLocalAvailabilityOfAssetPacks:completionHandler:`
- `BAAssetPackManager -ensureLocalAvailabilityOfAssetPacks:requireLatestVersions:completionHandler:`
- `BAAssetPackManager -fileDescriptorForPath:asLocalizedForLanguage:error:`
- `BAAssetPackManager -getLocallyAvailableLanguagesWithCompletionHandler:`
- `BAAssetPackManager -getManifestWithCompletionHandler:`
- `BAAssetPackManager -reconcilePreferredLanguagesWithCompletionHandler:`
- `BAAssetPackManager -resolvedLanguage`
- `BAAssetPackManager -setResolvedLanguage:`
- `BAAssetPackManifest -assetPackWithIdentifier:`
- `BAAssetPackManifest -availableLanguages`
- `BAAssetPackManifest -localizedAssetPacks`
- `BAAssetPackManifest -localizedAssetPacksForLanguage:`
- `BAAssetPackManifest -primaryLanguage`
- `BAAssetPackManifest -resolvedLanguage`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (7)

- `BAContentRequest.BAContentRequestLanguageChange`
- `BAManagedErrorCode.BAManagedErrorCodeLocalAvailabilityFailure`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Externs added (2)

- `BAFailuresErrorKey`
- `BASuccessesErrorKey`

## BackgroundTasks

### Methods added (1)

- `BGTaskScheduler -submitTaskRequest:completionHandler:`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## BrowserEngineCore

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## BrowserEngineKit

### Methods added (4)

- `BEProcessCapability +screenCaptureWithEnvironment:`
- `BEProcessCapability -activateWithError:`
- `BEProcessCapability -suspendWithError:`
- `BEWebContentFilter -evaluateURL:mainFrameURL:isMainFrame:completionHandler:`

### Enums added (7)

- `BEAccessibilityOrientation`
- `BEWebContentFilterPermissionDecision`
- `CGLCPContextPriorityRequest`
- `EvCmd`
- `NXMouseButton`
- `PMPageToPaperMappingType`
- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## BusinessChat

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CFOpenDirectory

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CalendarStore

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CallKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## Carbon

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CarbonCore

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## Cinematic

### Classes added (3)

- `CNAssetPreprocessConfiguration`
- `CNImageRenderingSession`
- `CNImageRenderingSessionConfiguration`

### Methods added (9)

- `CNAssetInfo +checkCinematicCapabilityForAsset:completionHandler:`
- `CNAssetInfo +defaultResourceDownloadTimeout`
- `CNAssetInfo +downloadResourcesForVersions:timeout:completionHandler:`
- `CNAssetInfo +resourceStatusForVersions:`
- `CNAssetInfo -cinematicCapability`
- `CNAssetInfo -downloadResourcesWithTimeout:completionHandler:`
- `CNAssetInfo -isPreprocessed`
- `CNAssetInfo -preprocessAssetWithConfiguration:completionHandler:`
- `CNAssetInfo -resourceStatus`

### Method signature changes (3)

- `CNRenderingSession -encodeRenderToCommandBuffer:frameAttributes:sourceImage:sourceDisparity:destinationImage:`
  - old: `(id<MTLCommandBuffer> _Nonnull, CNRenderingSessionFrameAttributes * _Nonnull, CVPixelBufferRef _Nonnull, CVPixelBufferRef _Nonnull, CVPixelBufferRef _Nonnull) → BOOL`
  - new: `(id<MTLCommandBuffer> _Nonnull, CNRenderingSessionFrameAttributes * _Nonnull, CVPixelBufferRef _Nonnull, CVPixelBufferRef _Nullable, CVPixelBufferRef _Nonnull) → BOOL`
- `CNRenderingSession -encodeRenderToCommandBuffer:frameAttributes:sourceImage:sourceDisparity:destinationLuma:destinationChroma:`
  - old: `(id<MTLCommandBuffer> _Nonnull, CNRenderingSessionFrameAttributes * _Nonnull, CVPixelBufferRef _Nonnull, CVPixelBufferRef _Nonnull, id<MTLTexture> _Nonnull, id<MTLTexture> _Nonnull) → BOOL`
  - new: `(id<MTLCommandBuffer> _Nonnull, CNRenderingSessionFrameAttributes * _Nonnull, CVPixelBufferRef _Nonnull, CVPixelBufferRef _Nullable, id<MTLTexture> _Nonnull, id<MTLTexture> _Nonnull) → BOOL`
- `CNRenderingSession -encodeRenderToCommandBuffer:frameAttributes:sourceImage:sourceDisparity:destinationRGBA:`
  - old: `(id<MTLCommandBuffer> _Nonnull, CNRenderingSessionFrameAttributes * _Nonnull, CVPixelBufferRef _Nonnull, CVPixelBufferRef _Nonnull, id<MTLTexture> _Nonnull) → BOOL`
  - new: `(id<MTLCommandBuffer> _Nonnull, CNRenderingSessionFrameAttributes * _Nonnull, CVPixelBufferRef _Nonnull, CVPixelBufferRef _Nullable, id<MTLTexture> _Nonnull) → BOOL`

### Enums added (4)

- `CNCinematicCapability`
- `CNCinematicResourceVersion`
- `CNResourceStatus`
- `task_shared_region_stubs_t`

### Enum members added (6)

- `CNCinematicErrorCode.CNCinematicErrorCodeDownloadFailed`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ClassKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CloudKit

### Methods added (1)

- `CKShareParticipant -isEqual:`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## Cocoa

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## Collaboration

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ColorSync

### Functions added (5)

- `ColorSyncProfileContainsHeadroomAdaptiveGainCurve`
- `ColorSyncProfileCopyHeadroomAdaptiveGainCurveInfoDictionary`
- `ColorSyncProfileCopyHeadroomAdaptiveGainCurveMetadata`
- `ColorSyncProfileCreateCopyWithHeadroomAdaptiveGainCurveInfoDictionary`
- `ColorSyncProfileCreateCopyWithHeadroomAdaptiveGainCurveMetadata`

### Externs added (25)

- `kColorSyncAlternateCurveCount`
- `kColorSyncAlternateCurveHeadroomStops`
- `kColorSyncAlternateGainCurveInfo`
- `kColorSyncBaselineHeadroomStops`
- `kColorSyncCoefficientBlue`
- `kColorSyncCoefficientComponent`
- `kColorSyncCoefficientGreen`
- `kColorSyncCoefficientMaxRGB`
- `kColorSyncCoefficientMinRGB`
- `kColorSyncCoefficientRed`
- `kColorSyncCommonComponentMixing`
- `kColorSyncCommonCurveParameters`
- `kColorSyncComponentCoefficients`
- `kColorSyncComponentMix`
- `kColorSyncControlPointSlopes`
- `kColorSyncControlPointsX`
- `kColorSyncControlPointsY`
- `kColorSyncCustomHDRReferenceWhite`
- `kColorSyncGainCurveChromaticities`
- `kColorSyncHeadroomAdaptiveGainCurveApplicationVersion`
- `kColorSyncHeadroomAdaptiveGainCurveColorVolumeTransform`
- `kColorSyncHeadroomAdaptiveGainCurveInfo`
- `kColorSyncHeadroomAdaptiveToneMappingInfo`
- `kColorSyncInterpolateSlopes`
- `kColorSyncMaxControlPointIndex`

## CommonPanels

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CompositorServices

### Enums added (9)

- `ar_accessory_anchor_tracking_state_t`
- `ar_accessory_chirality_t`
- `ar_accessory_source_type_t`
- `ar_accessory_tracking_error_code_t`
- `ar_camera_region_camera_enhancement_t`
- `ar_camera_region_error_code_t`
- `ar_transform_correction_t`
- `ar_world_anchor_sharing_availability_t`
- `task_shared_region_stubs_t`

### Enum members added (6)

- `ar_authorization_type_t.ar_authorization_type_accessory_tracking`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## Contacts

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ContactsUI

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CoreAudio

### Enum members added (8)

- `_anon_kAudioDevicePropertyConfigurationApplication.kAudioDevicePropertyWantsControlsRestored`
- `_anon_kAudioDevicePropertyConfigurationApplication.kAudioDevicePropertyWantsStreamFormatsRestored`
- `_anon_kAudioDevicePropertyPlugIn.kAudioDevicePropertySuggestedReferenceDevice`
- `_anon_kAudioDevicePropertyPlugIn.kAudioDevicePropertyVoiceActivityDetectionEnable`
- `_anon_kAudioDevicePropertyPlugIn.kAudioDevicePropertyVoiceActivityDetectionState`
- `_anon_kAudioDeviceTransportTypeAutoAggregate.kAudioDeviceTransportTypeContinuityCapture`
- `_anon_kAudioDeviceTransportTypeUnknown.kAudioDeviceTransportTypeRemoteScreen`
- `_anon_kAudioDeviceTransportTypeUnknown.kAudioDeviceTransportTypeRemoteStreaming`

### Enum members removed (5)

- `_anon_kAudioDevicePropertyJackIsConnected.kAudioDevicePropertyVoiceActivityDetectionEnable`
- `_anon_kAudioDevicePropertyJackIsConnected.kAudioDevicePropertyVoiceActivityDetectionState`
- `_anon_kAudioDevicePropertyJackIsConnected.kAudioDevicePropertyWantsControlsRestored`
- `_anon_kAudioDevicePropertyJackIsConnected.kAudioDevicePropertyWantsStreamFormatsRestored`
- `_anon_kAudioDeviceTransportTypeUnknown.kAudioDeviceTransportTypeContinuityCapture`

## CoreAudioKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CoreBluetooth

### Classes added (2)

- `CBChannelSoundingProcedureResults`
- `CBChannelSoundingSessionConfiguration`

### Methods added (2)

- `CBPeripheral -cancelChannelSoundingSession`
- `CBPeripheral -startChannelSoundingSession:`

### Enums added (2)

- `CBChannelSoundingSessionConfigurationRole`
- `task_shared_region_stubs_t`

### Enum members added (8)

- `CBCentralManagerFeature.CBCentralManagerFeatureChannelSounding`
- `CBError.CBErrorChannelSoundingConfigurationFailed`
- `CBError.CBErrorChannelSoundingProcedureFailed`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CoreData

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CoreGraphics

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (7)

- `CGPDFTagType.CGPDFTagTypeArtifact`
- `CGToneMapping.kCGToneMappingHeadroomAdaptiveGainCurve`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Structs added (2)

- `CGPDFMarkedContentItem`
- `CGPDFStructureElement`

### Functions added (18)

- `CGPDFContextAddStructureTreeRootChild`
- `CGPDFContextBeginMarkedContentSequence`
- `CGPDFContextBeginNonStructuralMarkedContentSequence`
- `CGPDFContextBeginObjectReference`
- `CGPDFContextEndMarkedContentSequence`
- `CGPDFContextEndObjectReference`
- `CGPDFMarkedContentItemRelease`
- `CGPDFMarkedContentItemRetain`
- `CGPDFStructureElementAddMarkedContentItem`
- `CGPDFStructureElementAddStructureElement`
- `CGPDFStructureElementCreate`
- `CGPDFStructureElementRelease`
- `CGPDFStructureElementRetain`
- `CGPDFStructureElementSetActualText`
- `CGPDFStructureElementSetAlternativeText`
- `CGPDFStructureElementSetExpansionText`
- `CGPDFStructureElementSetLanguageIdentifier`
- `CGPDFStructureElementSetTitle`

### Function signature changes (1)

- `CGDirectDisplayCopyCurrentMetalDevice`
  - old: `(CGDirectDisplayID) → NS_RETURNS_RETAINED id<MTLDevice>  _Nullable`
  - new: `(CGDirectDisplayID) → id<MTLDevice>  _Nullable`

## CoreImage

### Methods added (11)

- `CIContext -estimateRender:fromRect:toDestination:atPoint:error:`
- `CIImageProcessorKernel +applyWithTiledExtent:inputs:arguments:error:`
- `CIRAWFilter +supportedCameraModelsWithVersion:`
- `CIRAWFilter -despeckleAmount`
- `CIRAWFilter -downloadResourcesWithTimeout:completionHandler:`
- `CIRAWFilter -isDespeckleSupported`
- `CIRAWFilter -setDespeckleAmount:`
- `CIRenderTask -plannedPassCount`
- `CIRenderTask -plannedPeakMemory`
- `CIRenderTask -plannedPixelsOverdrawn`
- `CIRenderTask -plannedPixelsProcessed`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Externs added (3)

- `kCIImageSubsampleFactor`
- `kCIImageTypeIdentifierHint`
- `kCIImageUseHardwareAcceleration`

## CoreLocation

### Methods added (2)

- `CLLocationManager -headingBody`
- `CLLocationManager -setHeadingBody:`

### Protocols added (1)

- `CLBodyIdentifiable`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (7)

- `CLActivityType.CLActivityTypeMaritime`
- `CLLiveUpdateConfiguration.CLLiveUpdateConfigurationMaritime`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CoreMIDI

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CoreML

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CoreMedia

### Enums added (2)

- `_anon_kCMClockError_PreferredStartTimeNotAvailable`
- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Functions added (4)

- `CMClockCreateGenlockClock`
- `CMClockGetPreferredStartTimePattern`
- `CMClockImplementsGetPreferredStartTimePattern`
- `CMIsAnyDisplaySynchronizedToLockedGenlockSignal`

### Externs added (9)

- `kCMFormatDescriptionLogTransferFunction_AppleLog2`
- `kCMGenlockClockNotificationPayload_AnyDisplayIsSynchronizedToLockedGenlockSignal`
- `kCMGenlockClockNotification_DisplayGenlockModeChanged`
- `kCMMetadataBaseDataType_ISOLatin1`
- `kCMMetadataBaseDataType_MacRoman`
- `kCMMetadataDataType_QuickTimeMetadataSMPTE2094_50`
- `kCMMetadataFormatDescriptionKey_HumanReadableString`
- `kCMMetadataFormatDescriptionMetadataSpecificationKey_HumanReadableString`
- `kCMMetadataIdentifier_ITUT_T35MetadataSMPTE2094_50`

## CoreMediaIO

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CoreMotion

### Classes added (1)

- `CMRecordedDeviceMotion`

### Methods added (3)

- `CMDeviceMotion -headingAccuracy`
- `CMMotionManager -deviceMotionBody`
- `CMMotionManager -setDeviceMotionBody:`

### Protocols added (1)

- `CMBodyIdentifiable`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CoreServices

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CoreSpotlight

### Classes added (1)

- `CSSearchableIndexDescription`

### Methods added (1)

- `CSSearchableIndex -protectionClass`

### Method signature changes (26)

- `CSSearchableItemAttributeSet -actionIdentifiers`
  - old: `() → NSArray<NSString *> *`
  - new: `() → NSArray<NSString *> * _Nonnull`
- `CSSearchableItemAttributeSet -domainIdentifier`
  - old: `() → CS_AVAILABLE NSString *`
  - new: `() → NSString * _Nullable`
- `CSSearchableItemAttributeSet -isPriority`
  - old: `() → NSNumber *`
  - new: `() → NSNumber * _Nullable`
- `CSSearchableItemAttributeSet -isUserCreated`
  - old: `() → CS_AVAILABLE NSNumber *`
  - new: `() → NSNumber * _Nullable`
- `CSSearchableItemAttributeSet -isUserCurated`
  - old: `() → CS_AVAILABLE NSNumber *`
  - new: `() → NSNumber * _Nullable`
- `CSSearchableItemAttributeSet -isUserOwned`
  - old: `() → CS_AVAILABLE NSNumber *`
  - new: `() → NSNumber * _Nullable`
- `CSSearchableItemAttributeSet -providerDataTypeIdentifiers`
  - old: `() → CS_AVAILABLE NSArray<NSString *> *`
  - new: `() → NSArray<NSString *> * _Nullable`
- `CSSearchableItemAttributeSet -providerFileTypeIdentifiers`
  - old: `() → CS_AVAILABLE NSArray<NSString *> *`
  - new: `() → NSArray<NSString *> * _Nullable`
- `CSSearchableItemAttributeSet -providerInPlaceFileTypeIdentifiers`
  - old: `() → CS_AVAILABLE NSArray<NSString *> *`
  - new: `() → NSArray<NSString *> * _Nullable`
- `CSSearchableItemAttributeSet -rankingHint`
  - old: `() → CS_AVAILABLE NSNumber *`
  - new: `() → NSNumber * _Nullable`
- `CSSearchableItemAttributeSet -setActionIdentifiers:`
  - old: `(NSArray<NSString *> *) → void`
  - new: `(NSArray<NSString *> * _Nonnull) → void`
- `CSSearchableItemAttributeSet -setDomainIdentifier:`
  - old: `(CS_AVAILABLE NSString *) → void`
  - new: `(NSString * _Nullable) → void`
- `CSSearchableItemAttributeSet -setProviderDataTypeIdentifiers:`
  - old: `(CS_AVAILABLE NSArray<NSString *> *) → void`
  - new: `(NSArray<NSString *> * _Nullable) → void`
- `CSSearchableItemAttributeSet -setProviderFileTypeIdentifiers:`
  - old: `(CS_AVAILABLE NSArray<NSString *> *) → void`
  - new: `(NSArray<NSString *> * _Nullable) → void`
- `CSSearchableItemAttributeSet -setProviderInPlaceFileTypeIdentifiers:`
  - old: `(CS_AVAILABLE NSArray<NSString *> *) → void`
  - new: `(NSArray<NSString *> * _Nullable) → void`
- `CSSearchableItemAttributeSet -setRankingHint:`
  - old: `(CS_AVAILABLE NSNumber *) → void`
  - new: `(NSNumber * _Nullable) → void`
- `CSSearchableItemAttributeSet -setSharedItemContentType:`
  - old: `(UTType *) → void`
  - new: `(UTType * _Nullable) → void`
- `CSSearchableItemAttributeSet -setTranscribedTextContent:`
  - old: `(NSString *) → void`
  - new: `(NSString * _Nullable) → void`
- `CSSearchableItemAttributeSet -setUserCreated:`
  - old: `(CS_AVAILABLE NSNumber *) → void`
  - new: `(NSNumber * _Nullable) → void`
- `CSSearchableItemAttributeSet -setUserCurated:`
  - old: `(CS_AVAILABLE NSNumber *) → void`
  - new: `(NSNumber * _Nullable) → void`
- `CSSearchableItemAttributeSet -setUserOwned:`
  - old: `(CS_AVAILABLE NSNumber *) → void`
  - new: `(NSNumber * _Nullable) → void`
- `CSSearchableItemAttributeSet -setWeakRelatedUniqueIdentifier:`
  - old: `(CS_AVAILABLE NSString *) → void`
  - new: `(NSString * _Nullable) → void`
- `CSSearchableItemAttributeSet -sharedItemContentType`
  - old: `() → UTType *`
  - new: `() → UTType * _Nullable`
- `CSSearchableItemAttributeSet -textContentSummary`
  - old: `() → NSString *`
  - new: `() → NSString * _Nullable`
- `CSSearchableItemAttributeSet -transcribedTextContent`
  - old: `() → NSString *`
  - new: `() → NSString * _Nullable`
- `CSSearchableItemAttributeSet -weakRelatedUniqueIdentifier`
  - old: `() → CS_AVAILABLE NSString *`
  - new: `() → NSString * _Nullable`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CoreText

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CoreTransferable

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CoreVideo

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Function signature changes (1)

- `CVBufferCopyAttachment`
  - old: `(CVBufferRef _Nonnull, CFStringRef _Nonnull, CVAttachmentMode * _Nullable) → CF_RETURNS_RETAINED CFTypeRef`
  - new: `(CVBufferRef _Nonnull, CFStringRef _Nonnull, CVAttachmentMode * _Nullable) → CFTypeRef`

### Externs added (1)

- `kCVImageBufferHorizontalDisparityAdjustmentKey`

## CoreWLAN

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## CryptoTokenKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (6)

- `TKErrorCode.TKErrorCodeInvalidatedDeviceKey`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## DVDPlayback

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## DataDetection

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## DeviceCheck

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## DeviceDiscoveryExtension

### Method signature changes (4)

- `DDDevice -mediaContentSubtitle`
  - old: `() → NSString * _Nullable`
  - new: `() → NSString *`
- `DDDevice -mediaContentTitle`
  - old: `() → NSString * _Nullable`
  - new: `() → NSString *`
- `DDDevice -setMediaContentSubtitle:`
  - old: `(NSString * _Nullable) → void`
  - new: `(NSString *) → void`
- `DDDevice -setMediaContentTitle:`
  - old: `(NSString * _Nullable) → void`
  - new: `(NSString *) → void`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Enum base type changes (1)

- `nw_link_quality_t`
  - old: `int32`
  - new: `uint8`

## DirectoryService

### Functions removed (2)

- `dsIsDirServiceLocalRunning`
- `dsIsDirServiceRunning`

## DiscRecording

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## DiscRecordingUI

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## DiskArbitration

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## DockKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## DriverKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (7)

- `_anon_kIOMemoryMapFixedAddress.kIOMemoryMapCacheModePostedCombinedReordered`
- `_anon_kIORPCMessageRemote.kIORPCMessageDeepSerialization`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## EventKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ExceptionHandling

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ExecutionPolicy

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ExtensionKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ExternalAccessory

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## FSKit

### Classes added (31)

- `FSActivateResult`
- `FSBlockmapResult`
- `FSCheckAccessResult`
- `FSCompleteIOResult`
- `FSContext`
- `FSCreateFileKOIOResult`
- `FSCreateItemResult`
- `FSCreateLinkResult`
- `FSCreateSymlinkResult`
- `FSDeactivateItemResult`
- `FSEnumerateDirectoryResult`
- `FSFreeSpace`
- `FSGetAttributesResult`
- `FSGetXattrResult`
- `FSListXattrsResult`
- `FSLookupItemKOIOResult`
- `FSLookupItemResult`
- `FSOpenItemResult`
- `FSPreallocateKOIOResult`
- `FSPreallocateResult`
- `FSReadFileResult`
- `FSReadSymlinkResult`
- `FSRemoveItemResult`
- `FSRenameItemResult`
- `FSSeekRegionResult`
- `FSSetAttributesResult`
- `FSSetXattrResult`
- `FSUpgradeItemResult`
- `FSVolumeHandlerResult`
- `FSVolumeRenameResult`
- `FSWriteFileResult`

### Methods added (5)

- `FSClient -mountSingleVolumeForResource:bundleID:options:completionHandler:`
- `FSClient -openFileSystemExtensionsSettings`
- `FSEntityIdentifier -initWithUUID:qualifierData:`
- `FSItem -tryReclaimWithBlock:`
- `FSVolume -setCacheStateForItem:cacheMode:coherencyType:coherencyAction:`

### Protocols added (12)

- `FSVolumeAccessCheckHandler`
- `FSVolumeCommonOperations`
- `FSVolumeDataCacheHandler`
- `FSVolumeHandler`
- `FSVolumeItemDeactivationHandler`
- `FSVolumeKernelOffloadedIOHandler`
- `FSVolumeOpenCloseHandler`
- `FSVolumePreallocateHandler`
- `FSVolumeReadWriteHandler`
- `FSVolumeRenameHandler`
- `FSVolumeSeekRegionHandler`
- `FSVolumeXattrHandler`

### Enums added (6)

- `FSDataCacheErrorCode`
- `FSDataCacheMode`
- `FSKernelCacheCoherencyAction`
- `FSKernelCacheCoherencyType`
- `FSSeekRegion`
- `task_shared_region_stubs_t`

### Enum members added (6)

- `FSExtentType.FSExtentTypeReadOnly`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## FileProvider

### Method signature changes (2)

- `NSFileProviderExtension -documentStorageURL`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT NSURL *`
  - new: `() → NSURL *`
- `NSFileProviderExtension -providerIdentifier`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT NSString *`
  - new: `() → NSString *`

### Enums added (2)

- `NSFileProviderNamespacePolicy`
- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## FileProviderUI

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## FinderSync

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ForceFeedback

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## Foundation

### Methods added (2)

- `NSError +new`
- `NSError -init`

### Method signature changes (4)

- `NSTask -currentDirectoryPath`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT NSString *`
  - new: `() → NSString *`
- `NSTask -launchPath`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT NSString *`
  - new: `() → NSString *`
- `NSTask -setCurrentDirectoryPath:`
  - old: `(API_DEPRECATED_WITH_REPLACEMENT NSString *) → void`
  - new: `(NSString *) → void`
- `NSTask -setLaunchPath:`
  - old: `(API_DEPRECATED_WITH_REPLACEMENT NSString *) → void`
  - new: `(NSString *) → void`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Function signature changes (5)

- `CFBridgingRelease`
  - old: `(CFTypeRef _Nullable) → NS_RETURNS_RETAINED id  _Nullable`
  - new: `(CFTypeRef _Nullable) → id  _Nullable`
- `NSCopyMapTableWithZone`
  - old: `(NSMapTable * _Nonnull, NSZone * _Nullable) → NS_RETURNS_RETAINED NSMapTable * _Nonnull`
  - new: `(NSMapTable * _Nonnull, NSZone * _Nullable) → NSMapTable * _Nonnull`
- `NSCreateMapTable`
  - old: `(NSMapTableKeyCallBacks, NSMapTableValueCallBacks, NSUInteger) → NS_RETURNS_RETAINED NSMapTable * _Nonnull`
  - new: `(NSMapTableKeyCallBacks, NSMapTableValueCallBacks, NSUInteger) → NSMapTable * _Nonnull`
- `NSCreateMapTableWithZone`
  - old: `(NSMapTableKeyCallBacks, NSMapTableValueCallBacks, NSUInteger, NSZone * _Nullable) → NS_RETURNS_RETAINED NSMapTable * _Nonnull`
  - new: `(NSMapTableKeyCallBacks, NSMapTableValueCallBacks, NSUInteger, NSZone * _Nullable) → NSMapTable * _Nonnull`
- `NSMakeCollectable`
  - old: `(CF_CONSUMED CFTypeRef) → NS_RETURNS_RETAINED id  _Nullable | (CF_CONSUMED CFTypeRef) → NS_RETURNS_RETAINED id  _Nullable`
  - new: `(CF_CONSUMED CFTypeRef) → id  _Nullable | (CF_CONSUMED CFTypeRef) → id  _Nullable`

### Deprecation changes (3)

- `NSCalendar: deprecated 10.10 → (none)`
- `NSDateComponents: deprecated 10.10 → (none)`
- `NSURLHandle: deprecated 10.4 → (none)`

## GLKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## GameController

### Classes added (3)

- `GCControllerHomeButtonSettingsManager`
- `GCDeviceType`
- `GCSpatialAccessory`

### Methods removed (1)

- `GCDeviceHaptics -init`

### Enums added (5)

- `GCControllerHomeButtonSettingCustomizationStatus`
- `GCControllerHomeButtonSettingInAppAction`
- `GCControllerHomeButtonSettingSystemAction`
- `GCControllerHomeButtonSettingsCustomizationActivity`
- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Externs added (2)

- `GCSpatialAccessoryDidConnectNotification`
- `GCSpatialAccessoryDidDisconnectNotification`

## GameKit

### Method signature changes (1)

- `GKAchievementDescription -image`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT NSImage *`
  - new: `() → NSImage *`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## GameSave

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## GameplayKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## HIServices

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Function signature changes (16)

- `AXObserverCreate`
  - old: `(pid_t, AXObserverCallback _Nonnull, CF_RETURNS_RETAINED AXObserverRef  _Nullable *) → AXError`
  - new: `(pid_t, AXObserverCallback _Nonnull, AXObserverRef  _Nullable *) → AXError`
- `AXObserverCreateWithInfoCallback`
  - old: `(pid_t, AXObserverCallbackWithInfo _Nonnull, CF_RETURNS_RETAINED AXObserverRef  _Nullable *) → AXError`
  - new: `(pid_t, AXObserverCallbackWithInfo _Nonnull, AXObserverRef  _Nullable *) → AXError`
- `AXUIElementCopyActionDescription`
  - old: `(AXUIElementRef _Nonnull, CFStringRef _Nonnull, CF_RETURNS_RETAINED CFStringRef  _Nullable *) → AXError`
  - new: `(AXUIElementRef _Nonnull, CFStringRef _Nonnull, CFStringRef  _Nullable *) → AXError`
- `AXUIElementCopyActionNames`
  - old: `(AXUIElementRef _Nonnull, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → AXError`
  - new: `(AXUIElementRef _Nonnull, CFArrayRef  _Nullable *) → AXError`
- `AXUIElementCopyAttributeNames`
  - old: `(AXUIElementRef _Nonnull, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → AXError`
  - new: `(AXUIElementRef _Nonnull, CFArrayRef  _Nullable *) → AXError`
- `AXUIElementCopyAttributeValue`
  - old: `(AXUIElementRef _Nonnull, CFStringRef _Nonnull, CF_RETURNS_RETAINED CFTypeRef  _Nullable *) → AXError`
  - new: `(AXUIElementRef _Nonnull, CFStringRef _Nonnull, CFTypeRef  _Nullable *) → AXError`
- `AXUIElementCopyAttributeValues`
  - old: `(AXUIElementRef _Nonnull, CFStringRef _Nonnull, CFIndex, CFIndex, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → AXError`
  - new: `(AXUIElementRef _Nonnull, CFStringRef _Nonnull, CFIndex, CFIndex, CFArrayRef  _Nullable *) → AXError`
- `AXUIElementCopyElementAtPosition`
  - old: `(AXUIElementRef _Nonnull, float, float, CF_RETURNS_RETAINED AXUIElementRef  _Nullable *) → AXError`
  - new: `(AXUIElementRef _Nonnull, float, float, AXUIElementRef  _Nullable *) → AXError`
- `AXUIElementCopyMultipleAttributeValues`
  - old: `(AXUIElementRef _Nonnull, CFArrayRef _Nonnull, AXCopyMultipleAttributeOptions, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → AXError`
  - new: `(AXUIElementRef _Nonnull, CFArrayRef _Nonnull, AXCopyMultipleAttributeOptions, CFArrayRef  _Nullable *) → AXError`
- `AXUIElementCopyParameterizedAttributeNames`
  - old: `(AXUIElementRef _Nonnull, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → AXError`
  - new: `(AXUIElementRef _Nonnull, CFArrayRef  _Nullable *) → AXError`
- `AXUIElementCopyParameterizedAttributeValue`
  - old: `(AXUIElementRef _Nonnull, CFStringRef _Nonnull, CFTypeRef _Nonnull, CF_RETURNS_RETAINED CFTypeRef  _Nullable *) → AXError`
  - new: `(AXUIElementRef _Nonnull, CFStringRef _Nonnull, CFTypeRef _Nonnull, CFTypeRef  _Nullable *) → AXError`
- `PasteboardCopyItemFlavorData`
  - old: `(PasteboardRef _Nonnull, PasteboardItemID _Nonnull, CFStringRef _Nonnull, CF_RETURNS_RETAINED CFDataRef  _Nullable *) → OSStatus`
  - new: `(PasteboardRef _Nonnull, PasteboardItemID _Nonnull, CFStringRef _Nonnull, CFDataRef  _Nullable *) → OSStatus`
- `PasteboardCopyItemFlavors`
  - old: `(PasteboardRef _Nonnull, PasteboardItemID _Nonnull, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(PasteboardRef _Nonnull, PasteboardItemID _Nonnull, CFArrayRef  _Nullable *) → OSStatus`
- `PasteboardCopyName`
  - old: `(PasteboardRef _Nonnull, CF_RETURNS_RETAINED CFStringRef  _Nullable *) → OSStatus`
  - new: `(PasteboardRef _Nonnull, CFStringRef  _Nullable *) → OSStatus`
- `PasteboardCopyPasteLocation`
  - old: `(PasteboardRef _Nonnull, CF_RETURNS_RETAINED CFURLRef  _Nullable *) → OSStatus`
  - new: `(PasteboardRef _Nonnull, CFURLRef  _Nullable *) → OSStatus`
- `PasteboardCreate`
  - old: `(CFStringRef _Nullable, CF_RETURNS_RETAINED PasteboardRef  _Nullable *) → OSStatus`
  - new: `(CFStringRef _Nullable, PasteboardRef  _Nullable *) → OSStatus`

## HIToolbox

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## HealthKit

### Classes added (1)

- `HKLiveWorkoutZoneUpdate`

### Methods added (1)

- `HKHealthStore -getEarliestAuthorizedSampleDateForTypes:completion:`

### Method signature changes (2)

- `HKObject -source`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT HKSource *`
  - new: `() → HKSource *`
- `HKVerifiableClinicalRecord -JWSRepresentation`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT NSData *`
  - new: `() → NSData *`

### Enums added (2)

- `HKCategoryValueMenopausalState`
- `task_shared_region_stubs_t`

### Enum members added (7)

- `HKWorkoutActivityType.HKWorkoutActivityTypeGroup`
- `HKWorkoutActivityType.HKWorkoutActivityTypeRest`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Externs added (3)

- `HKCategoryTypeIdentifierBleedingAfterMenopause`
- `HKCategoryTypeIdentifierMenopausalState`
- `HKQuantityTypeIdentifierHeartRateVariabilityRMSSD`

## Help

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## Hypervisor

### Enums added (2)

- `_anon_HV_TLBI_OP_RVAE1IS`
- `hv_tlbi_op_t`

### Enum members added (16)

- `_anon_HV_FEATURE_REG_ID_AA64DFR0_EL1.HV_FEATURE_REG_ID_AA64ISAR2_EL1`
- `_anon_HV_FEATURE_REG_ID_AA64DFR0_EL1.HV_FEATURE_REG_ID_AA64MMFR3_EL1`
- `_anon_HV_FEATURE_REG_ID_AA64DFR0_EL1.HV_FEATURE_REG_ID_AA64MMFR4_EL1`
- `_anon_HV_FEATURE_REG_ID_AA64DFR0_EL1.HV_FEATURE_REG_ID_AA64PFR2_EL1`
- `_anon_HV_SYS_REG_DBGBVR0_EL1.HV_SYS_REG_ID_AA64ISAR2_EL1`
- `_anon_HV_SYS_REG_DBGBVR0_EL1.HV_SYS_REG_ID_AA64MMFR3_EL1`
- `_anon_HV_SYS_REG_DBGBVR0_EL1.HV_SYS_REG_ID_AA64MMFR4_EL1`
- `_anon_HV_SYS_REG_DBGBVR0_EL1.HV_SYS_REG_ID_AA64PFR2_EL1`
- `hv_feature_reg_t.HV_FEATURE_REG_ID_AA64ISAR2_EL1`
- `hv_feature_reg_t.HV_FEATURE_REG_ID_AA64MMFR3_EL1`
- `hv_feature_reg_t.HV_FEATURE_REG_ID_AA64MMFR4_EL1`
- `hv_feature_reg_t.HV_FEATURE_REG_ID_AA64PFR2_EL1`
- `hv_sys_reg_t.HV_SYS_REG_ID_AA64ISAR2_EL1`
- `hv_sys_reg_t.HV_SYS_REG_ID_AA64MMFR3_EL1`
- `hv_sys_reg_t.HV_SYS_REG_ID_AA64MMFR4_EL1`
- `hv_sys_reg_t.HV_SYS_REG_ID_AA64PFR2_EL1`

### Functions added (4)

- `hv_vcpu_get_serror`
- `hv_vcpu_get_wait_for_interrupt_time`
- `hv_vcpu_invalidate_tlb`
- `hv_vcpu_set_serror`

## ICADevices

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## IOBluetooth

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## IOBluetoothUI

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## IOKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Enum base type changes (10)

- `filesec_property_t`
  - old: `int64`
  - new: `int32`
- `idtype_t`
  - old: `int64`
  - new: `int32`
- `ipc_info_object_type_t`
  - old: `int64`
  - new: `uint32`
- `mach_vm_range_flags_t`
  - old: `int64`
  - new: `uint64`
- `mach_vm_range_flavor_t`
  - old: `int64`
  - new: `uint32`
- `mach_vm_range_tag_t`
  - old: `int64`
  - new: `uint16`
- `mpo_flags_t`
  - old: `int64`
  - new: `uint32`
- `os_clockid_t`
  - old: `int64`
  - new: `uint32`
- `ptrauth_key`
  - old: `int64`
  - new: `int32`
- `virtual_memory_guard_exception_code_t`
  - old: `int64`
  - new: `uint32`

## IOSurface

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## IOUSBHost

### Methods added (1)

- `IOUSBHostObject -dataWithCapacity:options:error:`

### Enums added (2)

- `IOUSBHostObjectDataOptions`
- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## IdentityLookup

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ImageCaptureCore

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ImageIO

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Externs added (2)

- `kCGImageSourceAllowableTypes`
- `kCGImageSourcePrioritizeQuality`

## ImageKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## InputMethodKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## InstallerPlugins

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## Intents

### Method signature changes (1)

- `INSetProfileInCarIntent -profileLabel`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT NSString *`
  - new: `() → NSString *`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## IntentsUI

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## JavaRuntimeSupport

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## JavaScriptCore

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## KernelManagement

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## LatentSemanticMapping

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## LaunchServices

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## LinkPresentation

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## LocalAuthentication

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## LocalAuthenticationEmbeddedUI

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## MLCompute

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## MPSCore

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (9)

- `MPSDataType.MPSDataTypeFloat4e2m1`
- `MPSDataType.MPSDataTypeFloat8e4m3`
- `MPSDataType.MPSDataTypeFloat8e5m2`
- `MPSDataType.MPSDataTypeFloat8e8m0`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## MPSImage

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## MPSMatrix

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## MPSNDArray

### Methods added (5)

- `MPSNDArrayIdentity -reshapeWithMTL4CommandEncoder:sourceArray:dimensionCount:dimensionSizes:destinationArray:`
- `MPSNDArrayIdentity -reshapeWithMTL4CommandEncoder:sourceArray:shape:destinationArray:`
- `MPSNDArrayIdentity -reshapeWithSourceArray:shape:`
- `MPSNDArrayMultiaryKernel -encodeWithMTL4CommandEncoder:sourceArrays:destinationArray:`
- `MPSNDArrayUnaryKernel -encodeWithMTL4CommandEncoder:sourceArray:destinationArray:`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## MPSNeuralNetwork

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## MPSRayIntersector

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## MailKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## MapKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Externs added (11)

- `MKPointOfInterestCategoryAirportTerminal`
- `MKPointOfInterestCategoryAutomotiveDealership`
- `MKPointOfInterestCategoryCommercialVehicleDealership`
- `MKPointOfInterestCategoryInformationBooth`
- `MKPointOfInterestCategoryMotorbikeDealership`
- `MKPointOfInterestCategoryPicnicArea`
- `MKPointOfInterestCategoryRangerStation`
- `MKPointOfInterestCategoryRestArea`
- `MKPointOfInterestCategoryScenicView`
- `MKPointOfInterestCategoryTicketOffice`
- `MKPointOfInterestCategoryVisitorCenter`

## Matter

### Classes added (145)

- `MTRAVAnalysisClusterActivateAnalysisStreamParams`
- `MTRAVAnalysisClusterAnalysisSessionEndEvent`
- `MTRAVAnalysisClusterAnalysisSessionStartEvent`
- `MTRAVAnalysisClusterAnalysisStreamStruct`
- `MTRAVAnalysisClusterContextTriggerStruct`
- `MTRAVAnalysisClusterDeactivateAnalysisStreamParams`
- `MTRAVAnalysisClusterDisableContextTriggersParams`
- `MTRAVAnalysisClusterEnableContextTriggersParams`
- `MTRAVAnalysisClusterEstablishAnalysisStreamParams`
- `MTRAVAnalysisClusterEstablishAnalysisStreamResponseParams`
- `MTRAVAnalysisClusterPerceivedContextEvent`
- `MTRAVAnalysisClusterRemoveAnalysisStreamParams`
- `MTRAVAnalysisClusterTrackedContext`
- `MTRAmbientContextSensingClusterAmbientContextDetectEndedEvent`
- `MTRAmbientContextSensingClusterAmbientContextDetectStartedEvent`
- `MTRAmbientContextSensingClusterAmbientContextTypeStruct`
- `MTRAmbientContextSensingClusterHoldTimeLimitsStruct`
- `MTRAmbientContextSensingClusterObjectCountConfigStruct`
- `MTRAmbientContextSensingClusterPredictedActivityStruct`
- `MTRAmbientSensingUnionClusterUnionContributorAddedEvent`
- `MTRAmbientSensingUnionClusterUnionContributorRemovedEvent`
- `MTRAmbientSensingUnionClusterUnionContributorStatusChangedEvent`
- `MTRAmbientSensingUnionClusterUnionContributorStruct`
- `MTRAudioControlClusterDecreaseVolumeParams`
- `MTRAudioControlClusterIncreaseVolumeParams`
- `MTRAudioControlClusterMuteParams`
- `MTRAudioControlClusterSetVolumeParams`
- `MTRAudioControlClusterToggleMutedParams`
- `MTRAudioControlClusterUnmuteParams`
- `MTRBaseClusterAVAnalysis`
- `MTRBaseClusterAmbientContextSensing`
- `MTRBaseClusterAmbientSensingUnion`
- `MTRBaseClusterAudioControl`
- `MTRBaseClusterCommissioningProxy`
- `MTRBaseClusterDynamicLighting`
- `MTRBaseClusterElectricalAlarm`
- `MTRBaseClusterElectricalDistribution`
- `MTRBaseClusterElectricalProtectionAlarm`
- `MTRBaseClusterHumidistat`
- `MTRBaseClusterMediaFileManagement`
- `MTRBaseClusterNetworkIdentityManagement`
- `MTRBaseClusterProximityRanging`
- `MTRBaseClusterSmokeConcentrationMeasurement`
- `MTRBaseClusterTemperatureControlledCabinetTopology`
- `MTRBaseClusterTestHiddenManufacturerSpecific`
- `MTRChimeClusterChimeStartedPlayingEvent`
- `MTRClosureControlClusterGroupedMoveToParams`
- `MTRClosureDimensionClusterGroupedSetTargetParams`
- `MTRClosureDimensionClusterGroupedStepParams`
- `MTRClusterAVAnalysis`
- `MTRClusterAmbientContextSensing`
- `MTRClusterAmbientSensingUnion`
- `MTRClusterAudioControl`
- `MTRClusterCommissioningProxy`
- `MTRClusterDynamicLighting`
- `MTRClusterElectricalAlarm`
- `MTRClusterElectricalDistribution`
- `MTRClusterElectricalProtectionAlarm`
- `MTRClusterHumidistat`
- `MTRClusterMediaFileManagement`
- `MTRClusterNetworkIdentityManagement`
- `MTRClusterProximityRanging`
- `MTRClusterSmokeConcentrationMeasurement`
- `MTRClusterTemperatureControlledCabinetTopology`
- `MTRClusterTestHiddenManufacturerSpecific`
- `MTRCommissioningProxyClusterProxyBackGroundScanStartRequestParams`
- `MTRCommissioningProxyClusterProxyBackGroundScanStopRequestParams`
- `MTRCommissioningProxyClusterProxyConnectRequestParams`
- `MTRCommissioningProxyClusterProxyConnectResponseParams`
- `MTRCommissioningProxyClusterProxyDisconnectRequestParams`
- `MTRCommissioningProxyClusterProxyMessageRequestParams`
- `MTRCommissioningProxyClusterProxyMessageResponseParams`
- `MTRCommissioningProxyClusterProxyScanRequestParams`
- `MTRCommissioningProxyClusterProxyScanResponseParams`
- `MTRCommissioningProxyClusterScanResultStruct`
- `MTRContentLauncherClusterContentAppInfo`
- `MTRContentLauncherClusterContentPresetStruct`
- `MTRContentLauncherClusterContentReplicationEvent`
- `MTRContentLauncherClusterContentReplicationRequestParams`
- `MTRContentLauncherClusterContentReplicationResponseParams`
- `MTRContentLauncherClusterLaunchUrlInfo`
- `MTRContentLauncherClusterPlayPresetParams`
- `MTRContentLauncherClusterReplicationInfo`
- `MTRDeviceEnergyManagementClusterCancelPowerRangeAdjustRequestParams`
- `MTRDeviceEnergyManagementClusterPowerRangeAdjustEndEvent`
- `MTRDeviceEnergyManagementClusterPowerRangeAdjustRequestParams`
- `MTRDeviceEnergyManagementClusterPowerRangeAdjustStartEvent`
- `MTRDeviceEnergyManagementClusterPowerRangeAdjustStruct`
- `MTRDynamicLightingClusterEffectColorStruct`
- `MTRDynamicLightingClusterEffectStruct`
- `MTRDynamicLightingClusterStartEffectParams`
- `MTRDynamicLightingClusterStopEffectParams`
- `MTRElectricalAlarmClusterModifyEnabledAlarmsParams`
- `MTRElectricalAlarmClusterNotifyEvent`
- `MTRElectricalAlarmClusterResetParams`
- `MTRElectricalAlarmClusterSetElectricalAlarmThresholdsParams`
- `MTRElectricalProtectionAlarmClusterArcFaultRatingsStruct`
- `MTRElectricalProtectionAlarmClusterModifyEnabledAlarmsParams`
- `MTRElectricalProtectionAlarmClusterNotifyEvent`
- `MTRElectricalProtectionAlarmClusterOverLoadRatingsStruct`
- `MTRElectricalProtectionAlarmClusterOverVoltageRatingsStruct`
- `MTRElectricalProtectionAlarmClusterResidualCurrentFaultRatingsStruct`
- `MTRElectricalProtectionAlarmClusterShortCircuitRatingsStruct`
- `MTRElectricalProtectionAlarmClusterSurgeProtectionRatingsStruct`
- `MTRGeneralDiagnosticsClusterDeviceLoadStruct`
- `MTRGroupKeyManagementClusterGroupcastAdoptionStruct`
- `MTRGroupcastClusterGroupcastTestingEvent`
- `MTRGroupcastClusterGroupcastTestingParams`
- `MTRHumidistatClusterSetSettingsParams`
- `MTRMediaFileManagementClusterAddFileParams`
- `MTRMediaFileManagementClusterAddFileResponseParams`
- `MTRMediaFileManagementClusterDeleteFileParams`
- `MTRMediaFileManagementClusterFileDescriptionStruct`
- `MTRMediaFileManagementClusterGetSharedFileParams`
- `MTRMediaFileManagementClusterGetSharedFileResponseParams`
- `MTRMediaFileManagementClusterOfferFileParams`
- `MTRMediaFileManagementClusterRequestSharedFilesParams`
- `MTRMediaFileManagementClusterSharedFilesAddedEvent`
- `MTRMediaPlaybackClusterContentInfoStruct`
- `MTRNetworkIdentityManagementClusterActiveNetworkIdentityStruct`
- `MTRNetworkIdentityManagementClusterAddClientParams`
- `MTRNetworkIdentityManagementClusterAddClientResponseParams`
- `MTRNetworkIdentityManagementClusterClientStruct`
- `MTRNetworkIdentityManagementClusterExportAdminSecretParams`
- `MTRNetworkIdentityManagementClusterExportAdminSecretResponseParams`
- `MTRNetworkIdentityManagementClusterImportAdminSecretParams`
- `MTRNetworkIdentityManagementClusterQueryIdentityParams`
- `MTRNetworkIdentityManagementClusterQueryIdentityResponseParams`
- `MTRNetworkIdentityManagementClusterRemoveClientParams`
- `MTROccupancySensingClusterPredictedOccupancyStruct`
- `MTRProximityRangingClusterBLERangingDeviceRoleConfigStruct`
- `MTRProximityRangingClusterBLTChannelSoundingDeviceRoleConfigStruct`
- `MTRProximityRangingClusterRDRStruct`
- `MTRProximityRangingClusterRangingCapabilitiesStruct`
- `MTRProximityRangingClusterRangingMeasurementDataStruct`
- `MTRProximityRangingClusterRangingResultEvent`
- `MTRProximityRangingClusterRangingSessionStatusEvent`
- `MTRProximityRangingClusterRangingTriggerConditionStruct`
- `MTRProximityRangingClusterReportingConditionStruct`
- `MTRProximityRangingClusterStartRangingRequestParams`
- `MTRProximityRangingClusterStartRangingResponseParams`
- `MTRProximityRangingClusterStopRangingRequestParams`
- `MTRProximityRangingClusterWiFiRangingDeviceRoleConfigStruct`
- `MTRPushAVStreamTransportClusterAudioStreamStruct`
- `MTRPushAVStreamTransportClusterVideoStreamStruct`

### Classes removed (7)

- `MTRBaseClusterTimer`
- `MTRClusterTimer`
- `MTRGroupcastClusterExpireGracePeriodParams`
- `MTRTimerClusterAddTimeParams`
- `MTRTimerClusterReduceTimeParams`
- `MTRTimerClusterResetTimerParams`
- `MTRTimerClusterSetTimerParams`

### Methods added (180)

- `MTRBaseClusterAppleDeviceInformation +readAttributeNeedsAdditionalConfigurationWithClusterStateCache:endpoint:queue:completion:`
- `MTRBaseClusterAppleDeviceInformation -readAttributeNeedsAdditionalConfigurationWithCompletion:`
- `MTRBaseClusterAppleDeviceInformation -subscribeAttributeNeedsAdditionalConfigurationWithParams:subscriptionEstablished:reportHandler:`
- `MTRBaseClusterBasicInformation +readAttributeDeviceLocationWithClusterStateCache:endpoint:queue:completion:`
- `MTRBaseClusterBasicInformation -readAttributeDeviceLocationWithCompletion:`
- `MTRBaseClusterBasicInformation -subscribeAttributeDeviceLocationWithParams:subscriptionEstablished:reportHandler:`
- `MTRBaseClusterBasicInformation -writeAttributeDeviceLocationWithValue:completion:`
- `MTRBaseClusterBasicInformation -writeAttributeDeviceLocationWithValue:params:completion:`
- `MTRBaseClusterBridgedDeviceBasicInformation +readAttributeDeviceLocationWithClusterStateCache:endpoint:queue:completion:`
- `MTRBaseClusterBridgedDeviceBasicInformation -readAttributeDeviceLocationWithCompletion:`
- `MTRBaseClusterBridgedDeviceBasicInformation -subscribeAttributeDeviceLocationWithParams:subscriptionEstablished:reportHandler:`
- `MTRBaseClusterBridgedDeviceBasicInformation -writeAttributeDeviceLocationWithValue:completion:`
- `MTRBaseClusterBridgedDeviceBasicInformation -writeAttributeDeviceLocationWithValue:params:completion:`
- `MTRBaseClusterCameraAVStreamManagement +readAttributeImageRotationDiscreteAnglesWithClusterStateCache:endpoint:queue:completion:`
- `MTRBaseClusterCameraAVStreamManagement -readAttributeImageRotationDiscreteAnglesWithCompletion:`
- `MTRBaseClusterCameraAVStreamManagement -subscribeAttributeImageRotationDiscreteAnglesWithParams:subscriptionEstablished:reportHandler:`
- `MTRBaseClusterCameraAVStreamManagement -writeAttributeImageRotationDiscreteAnglesWithValue:completion:`
- `MTRBaseClusterCameraAVStreamManagement -writeAttributeImageRotationDiscreteAnglesWithValue:params:completion:`
- `MTRBaseClusterClosureControl -groupedMoveToWithCompletion:`
- `MTRBaseClusterClosureControl -groupedMoveToWithParams:completion:`
- `MTRBaseClusterClosureDimension -groupedSetTargetWithCompletion:`
- `MTRBaseClusterClosureDimension -groupedSetTargetWithParams:completion:`
- `MTRBaseClusterClosureDimension -groupedStepWithParams:completion:`
- `MTRBaseClusterContentLauncher +readAttributeMovableWithClusterStateCache:endpoint:queue:completion:`
- `MTRBaseClusterContentLauncher +readAttributePresetsWithClusterStateCache:endpoint:queue:completion:`
- `MTRBaseClusterContentLauncher -contentReplicationRequestWithCompletion:`
- `MTRBaseClusterContentLauncher -contentReplicationRequestWithParams:completion:`
- `MTRBaseClusterContentLauncher -playPresetWithParams:completion:`
- `MTRBaseClusterContentLauncher -readAttributeMovableWithCompletion:`
- `MTRBaseClusterContentLauncher -readAttributePresetsWithCompletion:`
- `MTRBaseClusterContentLauncher -subscribeAttributeMovableWithParams:subscriptionEstablished:reportHandler:`
- `MTRBaseClusterContentLauncher -subscribeAttributePresetsWithParams:subscriptionEstablished:reportHandler:`
- `MTRBaseClusterDeviceEnergyManagement +readAttributePowerRangeAdjustmentWithClusterStateCache:endpoint:queue:completion:`
- `MTRBaseClusterDeviceEnergyManagement -cancelPowerRangeAdjustRequestWithCompletion:`
- `MTRBaseClusterDeviceEnergyManagement -cancelPowerRangeAdjustRequestWithParams:completion:`
- `MTRBaseClusterDeviceEnergyManagement -powerRangeAdjustRequestWithParams:completion:`
- `MTRBaseClusterDeviceEnergyManagement -readAttributePowerRangeAdjustmentWithCompletion:`
- `MTRBaseClusterDeviceEnergyManagement -subscribeAttributePowerRangeAdjustmentWithParams:subscriptionEstablished:reportHandler:`
- `MTRBaseClusterGeneralDiagnostics +readAttributeDeviceLoadStatusWithClusterStateCache:endpoint:queue:completion:`
- `MTRBaseClusterGeneralDiagnostics -readAttributeDeviceLoadStatusWithCompletion:`
- `MTRBaseClusterGeneralDiagnostics -subscribeAttributeDeviceLoadStatusWithParams:subscriptionEstablished:reportHandler:`
- `MTRBaseClusterGroupKeyManagement +readAttributeGroupcastAdoptionWithClusterStateCache:endpoint:queue:completion:`
- `MTRBaseClusterGroupKeyManagement -readAttributeGroupcastAdoptionWithParams:completion:`
- `MTRBaseClusterGroupKeyManagement -subscribeAttributeGroupcastAdoptionWithParams:subscriptionEstablished:reportHandler:`
- `MTRBaseClusterGroupKeyManagement -writeAttributeGroupcastAdoptionWithValue:completion:`
- `MTRBaseClusterGroupKeyManagement -writeAttributeGroupcastAdoptionWithValue:params:completion:`
- `MTRBaseClusterGroupcast +readAttributeFabricUnderTestWithClusterStateCache:endpoint:queue:completion:`
- `MTRBaseClusterGroupcast +readAttributeMaxMcastAddrCountWithClusterStateCache:endpoint:queue:completion:`
- `MTRBaseClusterGroupcast +readAttributeUsedMcastAddrCountWithClusterStateCache:endpoint:queue:completion:`
- `MTRBaseClusterGroupcast -groupcastTestingWithParams:completion:`
- `MTRBaseClusterGroupcast -readAttributeFabricUnderTestWithCompletion:`
- `MTRBaseClusterGroupcast -readAttributeMaxMcastAddrCountWithCompletion:`
- `MTRBaseClusterGroupcast -readAttributeUsedMcastAddrCountWithCompletion:`
- `MTRBaseClusterGroupcast -subscribeAttributeFabricUnderTestWithParams:subscriptionEstablished:reportHandler:`
- `MTRBaseClusterGroupcast -subscribeAttributeMaxMcastAddrCountWithParams:subscriptionEstablished:reportHandler:`
- `MTRBaseClusterGroupcast -subscribeAttributeUsedMcastAddrCountWithParams:subscriptionEstablished:reportHandler:`
- `MTRBaseClusterMediaPlayback +readAttributeAvailableCommandsWithClusterStateCache:endpoint:queue:completion:`
- `MTRBaseClusterMediaPlayback +readAttributeContentInfoWithClusterStateCache:endpoint:queue:completion:`
- `MTRBaseClusterMediaPlayback -readAttributeAvailableCommandsWithCompletion:`
- `MTRBaseClusterMediaPlayback -readAttributeContentInfoWithCompletion:`
- `MTRBaseClusterMediaPlayback -subscribeAttributeAvailableCommandsWithParams:subscriptionEstablished:reportHandler:`
- `MTRBaseClusterMediaPlayback -subscribeAttributeContentInfoWithParams:subscriptionEstablished:reportHandler:`
- `MTRBaseClusterOccupancySensing +readAttributePredictedOccupancyWithClusterStateCache:endpoint:queue:completion:`
- `MTRBaseClusterOccupancySensing -readAttributePredictedOccupancyWithCompletion:`
- `MTRBaseClusterOccupancySensing -subscribeAttributePredictedOccupancyWithParams:subscriptionEstablished:reportHandler:`
- `MTRBaseClusterOperationalState +readAttributeAppleOperationCompletedWithClusterStateCache:endpoint:queue:completion:`
- `MTRBaseClusterOperationalState +readAttributeAppleOperationStartTimeWithClusterStateCache:endpoint:queue:completion:`
- `MTRBaseClusterOperationalState -readAttributeAppleOperationCompletedWithCompletion:`
- `MTRBaseClusterOperationalState -readAttributeAppleOperationStartTimeWithCompletion:`
- `MTRBaseClusterOperationalState -subscribeAttributeAppleOperationCompletedWithParams:subscriptionEstablished:reportHandler:`
- `MTRBaseClusterOperationalState -subscribeAttributeAppleOperationStartTimeWithParams:subscriptionEstablished:reportHandler:`
- `MTRBaseClusterSmokeCOAlarm +readAttributeUnmountedWithClusterStateCache:endpoint:queue:completion:`
- `MTRBaseClusterSmokeCOAlarm -readAttributeUnmountedWithCompletion:`
- `MTRBaseClusterSmokeCOAlarm -subscribeAttributeUnmountedWithParams:subscriptionEstablished:reportHandler:`
- `MTRBasicInformationClusterCapabilityMinimaStruct -readPathsSupported`
- `MTRBasicInformationClusterCapabilityMinimaStruct -setReadPathsSupported:`
- `MTRBasicInformationClusterCapabilityMinimaStruct -setSimultaneousInvocationsSupported:`
- `MTRBasicInformationClusterCapabilityMinimaStruct -setSimultaneousWritesSupported:`
- `MTRBasicInformationClusterCapabilityMinimaStruct -setSubscribePathsSupported:`
- `MTRBasicInformationClusterCapabilityMinimaStruct -simultaneousInvocationsSupported`
- `MTRBasicInformationClusterCapabilityMinimaStruct -simultaneousWritesSupported`
- `MTRBasicInformationClusterCapabilityMinimaStruct -subscribePathsSupported`
- `MTRChimeClusterPlayChimeSoundParams -chimeID`
- `MTRChimeClusterPlayChimeSoundParams -setChimeID:`
- `MTRClusterAppleDeviceInformation -readAttributeNeedsAdditionalConfigurationWithParams:`
- `MTRClusterBasicInformation -readAttributeDeviceLocationWithParams:`
- `MTRClusterBasicInformation -writeAttributeDeviceLocationWithValue:expectedValueInterval:`
- `MTRClusterBasicInformation -writeAttributeDeviceLocationWithValue:expectedValueInterval:params:`
- `MTRClusterBridgedDeviceBasicInformation -readAttributeDeviceLocationWithParams:`
- `MTRClusterBridgedDeviceBasicInformation -writeAttributeDeviceLocationWithValue:expectedValueInterval:`
- `MTRClusterBridgedDeviceBasicInformation -writeAttributeDeviceLocationWithValue:expectedValueInterval:params:`
- `MTRClusterCameraAVStreamManagement -readAttributeImageRotationDiscreteAnglesWithParams:`
- `MTRClusterCameraAVStreamManagement -writeAttributeImageRotationDiscreteAnglesWithValue:expectedValueInterval:`
- `MTRClusterCameraAVStreamManagement -writeAttributeImageRotationDiscreteAnglesWithValue:expectedValueInterval:params:`
- `MTRClusterClosureControl -groupedMoveToWithExpectedValues:expectedValueInterval:completion:`
- `MTRClusterClosureControl -groupedMoveToWithParams:expectedValues:expectedValueInterval:completion:`
- `MTRClusterClosureDimension -groupedSetTargetWithExpectedValues:expectedValueInterval:completion:`
- `MTRClusterClosureDimension -groupedSetTargetWithParams:expectedValues:expectedValueInterval:completion:`
- `MTRClusterClosureDimension -groupedStepWithParams:expectedValues:expectedValueInterval:completion:`
- `MTRClusterContentLauncher -contentReplicationRequestWithExpectedValues:expectedValueInterval:completion:`
- `MTRClusterContentLauncher -contentReplicationRequestWithParams:expectedValues:expectedValueInterval:completion:`
- `MTRClusterContentLauncher -playPresetWithParams:expectedValues:expectedValueInterval:completion:`
- `MTRClusterContentLauncher -readAttributeMovableWithParams:`
- `MTRClusterContentLauncher -readAttributePresetsWithParams:`
- `MTRClusterDeviceEnergyManagement -cancelPowerRangeAdjustRequestWithExpectedValues:expectedValueInterval:completion:`
- `MTRClusterDeviceEnergyManagement -cancelPowerRangeAdjustRequestWithParams:expectedValues:expectedValueInterval:completion:`
- `MTRClusterDeviceEnergyManagement -powerRangeAdjustRequestWithParams:expectedValues:expectedValueInterval:completion:`
- `MTRClusterDeviceEnergyManagement -readAttributePowerRangeAdjustmentWithParams:`
- `MTRClusterGeneralDiagnostics -readAttributeDeviceLoadStatusWithParams:`
- `MTRClusterGroupKeyManagement -readAttributeGroupcastAdoptionWithParams:`
- `MTRClusterGroupKeyManagement -writeAttributeGroupcastAdoptionWithValue:expectedValueInterval:`
- `MTRClusterGroupKeyManagement -writeAttributeGroupcastAdoptionWithValue:expectedValueInterval:params:`
- `MTRClusterGroupcast -groupcastTestingWithParams:expectedValues:expectedValueInterval:completion:`
- `MTRClusterGroupcast -readAttributeFabricUnderTestWithParams:`
- `MTRClusterGroupcast -readAttributeMaxMcastAddrCountWithParams:`
- `MTRClusterGroupcast -readAttributeUsedMcastAddrCountWithParams:`
- `MTRClusterMediaPlayback -readAttributeAvailableCommandsWithParams:`
- `MTRClusterMediaPlayback -readAttributeContentInfoWithParams:`
- `MTRClusterOccupancySensing -readAttributePredictedOccupancyWithParams:`
- `MTRClusterOperationalState -readAttributeAppleOperationCompletedWithParams:`
- `MTRClusterOperationalState -readAttributeAppleOperationStartTimeWithParams:`
- `MTRClusterSmokeCOAlarm -readAttributeUnmountedWithParams:`
- `MTRCommodityTariffClusterTariffComponentStruct -externalID`
- `MTRCommodityTariffClusterTariffComponentStruct -setExternalID:`
- `MTRContentLauncherClusterLaunchContentParams -contentAppProductID`
- `MTRContentLauncherClusterLaunchContentParams -contentAppVendorID`
- `MTRContentLauncherClusterLaunchContentParams -setContentAppProductID:`
- `MTRContentLauncherClusterLaunchContentParams -setContentAppVendorID:`
- `MTRContentLauncherClusterLaunchURLParams -contentHeaders`
- `MTRContentLauncherClusterLaunchURLParams -contentType`
- `MTRContentLauncherClusterLaunchURLParams -nextUrl`
- `MTRContentLauncherClusterLaunchURLParams -offsetMillisecs`
- `MTRContentLauncherClusterLaunchURLParams -playbackPreferences`
- `MTRContentLauncherClusterLaunchURLParams -queueType`
- `MTRContentLauncherClusterLaunchURLParams -setContentHeaders:`
- `MTRContentLauncherClusterLaunchURLParams -setContentType:`
- `MTRContentLauncherClusterLaunchURLParams -setNextUrl:`
- `MTRContentLauncherClusterLaunchURLParams -setOffsetMillisecs:`
- `MTRContentLauncherClusterLaunchURLParams -setPlaybackPreferences:`
- `MTRContentLauncherClusterLaunchURLParams -setQueueType:`
- `MTRDataTypeWebRTCSessionStruct -audioStreams`
- `MTRDataTypeWebRTCSessionStruct -setAudioStreams:`
- `MTRDataTypeWebRTCSessionStruct -setVideoStreams:`
- `MTRDataTypeWebRTCSessionStruct -videoStreams`
- `MTRGroupcastClusterJoinGroupParams -keySetID`
- `MTRGroupcastClusterJoinGroupParams -mcastAddrPolicy`
- `MTRGroupcastClusterJoinGroupParams -replaceEndpoints`
- `MTRGroupcastClusterJoinGroupParams -setKeySetID:`
- `MTRGroupcastClusterJoinGroupParams -setMcastAddrPolicy:`
- `MTRGroupcastClusterJoinGroupParams -setReplaceEndpoints:`
- `MTRGroupcastClusterMembershipStruct -keySetID`
- `MTRGroupcastClusterMembershipStruct -mcastAddrPolicy`
- `MTRGroupcastClusterMembershipStruct -setKeySetID:`
- `MTRGroupcastClusterMembershipStruct -setMcastAddrPolicy:`
- `MTRGroupcastClusterUpdateGroupKeyParams -keySetID`
- `MTRGroupcastClusterUpdateGroupKeyParams -setKeySetID:`
- `MTRJointFabricAdministratorClusterICACCSRResponseParams -setStatusCode:`
- `MTRJointFabricAdministratorClusterICACCSRResponseParams -statusCode`
- `MTRMediaPlaybackClusterTrackAttributesStruct -characteristics`
- `MTRMediaPlaybackClusterTrackAttributesStruct -setCharacteristics:`
- `MTRPushAVStreamTransportClusterPushTransportBeginEvent -cmafSessionNumber`
- `MTRPushAVStreamTransportClusterPushTransportBeginEvent -containerType`
- `MTRPushAVStreamTransportClusterPushTransportBeginEvent -setCmafSessionNumber:`
- `MTRPushAVStreamTransportClusterPushTransportBeginEvent -setContainerType:`
- `MTRPushAVStreamTransportClusterPushTransportEndEvent -cmafSessionNumber`
- `MTRPushAVStreamTransportClusterPushTransportEndEvent -containerType`
- `MTRPushAVStreamTransportClusterPushTransportEndEvent -setCmafSessionNumber:`
- `MTRPushAVStreamTransportClusterPushTransportEndEvent -setContainerType:`
- `MTRPushAVStreamTransportClusterTransportOptionsStruct -audioStreams`
- `MTRPushAVStreamTransportClusterTransportOptionsStruct -setAudioStreams:`
- `MTRPushAVStreamTransportClusterTransportOptionsStruct -setVideoStreams:`
- `MTRPushAVStreamTransportClusterTransportOptionsStruct -videoStreams`
- `MTRWebRTCTransportProviderClusterProvideOfferParams -audioStreams`
- `MTRWebRTCTransportProviderClusterProvideOfferParams -setAudioStreams:`
- `MTRWebRTCTransportProviderClusterProvideOfferParams -setVideoStreams:`
- `MTRWebRTCTransportProviderClusterProvideOfferParams -videoStreams`
- `MTRWebRTCTransportProviderClusterSolicitOfferParams -audioStreams`
- `MTRWebRTCTransportProviderClusterSolicitOfferParams -setAudioStreams:`
- `MTRWebRTCTransportProviderClusterSolicitOfferParams -setVideoStreams:`
- `MTRWebRTCTransportProviderClusterSolicitOfferParams -videoStreams`

### Methods removed (24)

- `MTRBaseClusterGroupcast -expireGracePeriodWithParams:completion:`
- `MTRClusterGroupcast -expireGracePeriodWithParams:expectedValues:expectedValueInterval:completion:`
- `MTRGroupcastClusterJoinGroupParams -gracePeriod`
- `MTRGroupcastClusterJoinGroupParams -keyID`
- `MTRGroupcastClusterJoinGroupParams -setGracePeriod:`
- `MTRGroupcastClusterJoinGroupParams -setKeyID:`
- `MTRGroupcastClusterLeaveGroupResponseParams -listTooLarge`
- `MTRGroupcastClusterLeaveGroupResponseParams -setListTooLarge:`
- `MTRGroupcastClusterMembershipStruct -expiringKeyID`
- `MTRGroupcastClusterMembershipStruct -keyID`
- `MTRGroupcastClusterMembershipStruct -setExpiringKeyID:`
- `MTRGroupcastClusterMembershipStruct -setKeyID:`
- `MTRGroupcastClusterUpdateGroupKeyParams -gracePeriod`
- `MTRGroupcastClusterUpdateGroupKeyParams -keyID`
- `MTRGroupcastClusterUpdateGroupKeyParams -setGracePeriod:`
- `MTRGroupcastClusterUpdateGroupKeyParams -setKeyID:`
- `MTRJointFabricDatastoreClusterDatastoreEndpointEntryStruct -setStatusEntry:`
- `MTRJointFabricDatastoreClusterDatastoreEndpointEntryStruct -statusEntry`
- `MTRJointFabricDatastoreClusterDatastoreGroupKeySetStruct -groupKeyMulticastPolicy`
- `MTRJointFabricDatastoreClusterDatastoreGroupKeySetStruct -setGroupKeyMulticastPolicy:`
- `MTRPushAVStreamTransportClusterCMAFContainerOptionsStruct -cencKey`
- `MTRPushAVStreamTransportClusterCMAFContainerOptionsStruct -cencKeyID`
- `MTRPushAVStreamTransportClusterCMAFContainerOptionsStruct -setCencKey:`
- `MTRPushAVStreamTransportClusterCMAFContainerOptionsStruct -setCencKeyID:`

### Method signature changes (16)

- `MTRGroupcastClusterLeaveGroupResponseParams -endpoints`
  - old: `() → NSArray * _Nullable`
  - new: `() → NSArray * _Nonnull`
- `MTRGroupcastClusterLeaveGroupResponseParams -setEndpoints:`
  - old: `(NSArray * _Nullable) → void`
  - new: `(NSArray * _Nonnull) → void`
- `MTRGroupcastClusterMembershipStruct -endpoints`
  - old: `() → NSArray * _Nonnull`
  - new: `() → NSArray * _Nullable`
- `MTRGroupcastClusterMembershipStruct -hasAuxiliaryACL`
  - old: `() → NSNumber * _Nonnull`
  - new: `() → NSNumber * _Nullable`
- `MTRGroupcastClusterMembershipStruct -setEndpoints:`
  - old: `(NSArray * _Nonnull) → void`
  - new: `(NSArray * _Nullable) → void`
- `MTRGroupcastClusterMembershipStruct -setHasAuxiliaryACL:`
  - old: `(NSNumber * _Nonnull) → void`
  - new: `(NSNumber * _Nullable) → void`
- `MTRJointFabricAdministratorClusterICACCSRResponseParams -icaccsr`
  - old: `() → NSData * _Nonnull`
  - new: `() → NSData * _Nullable`
- `MTRJointFabricAdministratorClusterICACCSRResponseParams -setIcaccsr:`
  - old: `(NSData * _Nonnull) → void`
  - new: `(NSData * _Nullable) → void`
- `MTRJointFabricDatastoreClusterUpdateAdminParams -nodeID`
  - old: `() → NSNumber * _Nullable`
  - new: `() → NSNumber * _Nonnull`
- `MTRJointFabricDatastoreClusterUpdateAdminParams -setNodeID:`
  - old: `(NSNumber * _Nullable) → void`
  - new: `(NSNumber * _Nonnull) → void`
- `MTRJointFabricDatastoreClusterUpdateGroupParams -groupPermission`
  - old: `() → NSNumber * _Nonnull`
  - new: `() → NSNumber * _Nullable`
- `MTRJointFabricDatastoreClusterUpdateGroupParams -setGroupPermission:`
  - old: `(NSNumber * _Nonnull) → void`
  - new: `(NSNumber * _Nullable) → void`
- `MTRPushAVStreamTransportClusterCMAFContainerOptionsStruct -sessionGroup`
  - old: `() → NSNumber * _Nonnull`
  - new: `() → NSNumber * _Nullable`
- `MTRPushAVStreamTransportClusterCMAFContainerOptionsStruct -setSessionGroup:`
  - old: `(NSNumber * _Nonnull) → void`
  - new: `(NSNumber * _Nullable) → void`
- `MTRPushAVStreamTransportClusterCMAFContainerOptionsStruct -setTrackName:`
  - old: `(NSString * _Nonnull) → void`
  - new: `(NSString * _Nullable) → void`
- `MTRPushAVStreamTransportClusterCMAFContainerOptionsStruct -trackName`
  - old: `() → NSString * _Nonnull`
  - new: `() → NSString * _Nullable`

### Enums added (79)

- `MTRAVAnalysisAnalysisStreamState`
- `MTRAVAnalysisFeature`
- `MTRAmbientContextSensingFeature`
- `MTRAmbientSensingUnionUnionContributorStatus`
- `MTRAmbientSensingUnionUnionHealth`
- `MTRAppleAccessoryConnectivityMonitorAppleRecoveryConfiguration`
- `MTRAppleAccessoryConnectivityMonitorFeature`
- `MTRAppleAccessoryErrorsAppleAccessoryErrorCategory`
- `MTRAppleAccessoryErrorsAppleAccessoryErrorSeverity`
- `MTRAppleAccessoryErrorsAppleAccessoryPersistentErrorCode`
- `MTRAppleAccessoryErrorsAppleAccessoryTransientErrorCode`
- `MTRAppleBluetoothDiagnosticsAppleBluetoothConnectionStatus`
- `MTRAppleBluetoothDiagnosticsAppleBluetoothVersion`
- `MTRAppleBluetoothDiagnosticsFeature`
- `MTRAppleIntelligentStartAppleFinishAtTimestampChangeSource`
- `MTRAppleLaundryCombinationWasherDryerControlsAppleWasherDryerCycle`
- `MTRAppleSaturationDiagnosticsFeature`
- `MTRAppleStabilityDiagnosticsAppleBootReason`
- `MTRAppleStabilityDiagnosticsAppleHardwareFault`
- `MTRAppleThermalDiagnosticsFeature`
- `MTRAudioControlFeature`
- `MTRAudioControlUnmutePolicy`
- `MTRAudioControlUnmuteVolume`
- `MTRBooleanStateFeature`
- `MTRCommissioningProxyCapabilitiesBitmap`
- `MTRCommissioningProxyFeature`
- `MTRCommissioningProxyWiFiBandBitmap`
- `MTRContentLauncherQueueType`
- `MTRDoorLockStatusCode`
- `MTRDynamicLightingEffectColorMode`
- `MTRDynamicLightingEffectSource`
- `MTRElectricalAlarmAlarmBitmap`
- `MTRElectricalAlarmFeature`
- `MTRElectricalDistributionEndOfLife`
- `MTRElectricalProtectionAlarmAlarmBitmap`
- `MTRElectricalProtectionAlarmArcCauseBitmap`
- `MTRElectricalProtectionAlarmCurrentTripCurve`
- `MTRElectricalProtectionAlarmCurrentTripMechanismBitmap`
- `MTRElectricalProtectionAlarmCurrentWaveform`
- `MTRElectricalProtectionAlarmFeature`
- `MTRElectricalProtectionAlarmGroundFaultClass`
- `MTRElectricalProtectionAlarmSurgeProtectionClassBitmap`
- `MTRElectricalProtectionAlarmSurgeProtectionTypeBitmap`
- `MTRElectricalProtectionAlarmTrippingCharacteristicsBitmap`
- `MTRElectricalProtectionAlarmVoltageTripMechanismBitmap`
- `MTRGroupKeyManagementGroupKeyMulticastPolicy`
- `MTRGroupcastMulticastAddrPolicy`
- `MTRGroupcastTestResult`
- `MTRGroupcastTesting`
- `MTRHumidistatFeature`
- `MTRHumidistatMistTypeBitmap`
- `MTRHumidistatMode`
- `MTRHumidistatSystemState`
- `MTRJointFabricAdministratorICACCSRResponseStatusCode`
- `MTRMediaFileManagementFeature`
- `MTRMediaFileManagementFileStatus`
- `MTRMediaPlaybackMediaType`
- `MTRNetworkIdentityManagementIdentityType`
- `MTRProximityRangingBLTCSMode`
- `MTRProximityRangingBLTCSSecurityLevel`
- `MTRProximityRangingFeature`
- `MTRProximityRangingNADM`
- `MTRProximityRangingRDRReference`
- `MTRProximityRangingRadioBandBitmap`
- `MTRProximityRangingRangingBandwidthBitmap`
- `MTRProximityRangingRangingRole`
- `MTRProximityRangingRangingSecurity`
- `MTRProximityRangingRangingSessionStatus`
- `MTRProximityRangingRangingTech`
- `MTRProximityRangingResultCode`
- `MTRSmokeConcentrationMeasurementFeature`
- `MTRSmokeConcentrationMeasurementLevelValue`
- `MTRSmokeConcentrationMeasurementMeasurementMedium`
- `MTRSmokeConcentrationMeasurementMeasurementUnit`
- `MTRTemperatureControlledCabinetTopologyTopology`
- `MTRWiFiNetworkManagementAppleFeatureBitmap`
- `MTRWiFiNetworkManagementAppleStatusCode`
- `MTRWiFiNetworkManagementAppleWiFiSecurityMode`
- `task_shared_region_stubs_t`

### Enums removed (3)

- `MTRJointFabricAdministratorStatusCode`
- `MTRTimerFeature`
- `MTRTimerStatus`

### Enum members added (370)

- `MTRAttributeIDType.MTRAttributeIDTypeClusterAVAnalysisAttributeAcceptedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAVAnalysisAttributeActiveAmbientContextTriggersID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAVAnalysisAttributeAnalysisStreamsID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAVAnalysisAttributeAttributeListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAVAnalysisAttributeClusterRevisionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAVAnalysisAttributeCurrentAnalysisStreamCountID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAVAnalysisAttributeFeatureMapID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAVAnalysisAttributeGeneratedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAVAnalysisAttributeMaxAnalysisStreamCountID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAVAnalysisAttributeSupportedAmbientContextsID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAVAnalysisAttributeTrackingEnabledID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientContextSensingAttributeAcceptedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientContextSensingAttributeAmbientContextTypeID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientContextSensingAttributeAmbientContextTypeSupportedID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientContextSensingAttributeAttributeListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientContextSensingAttributeAudioContextDetectedID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientContextSensingAttributeClusterRevisionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientContextSensingAttributeFeatureMapID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientContextSensingAttributeGeneratedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientContextSensingAttributeHoldTimeID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientContextSensingAttributeHoldTimeLimitsID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientContextSensingAttributeHumanActivityDetectedID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientContextSensingAttributeObjectCountConfigID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientContextSensingAttributeObjectCountID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientContextSensingAttributeObjectCountThresholdReachedID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientContextSensingAttributeObjectIdentifiedID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientContextSensingAttributePredictedActivityID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientContextSensingAttributeSensorFusionSupportedID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientContextSensingAttributeSimultaneousDetectionLimitID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientSensingUnionAttributeAcceptedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientSensingUnionAttributeAttributeListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientSensingUnionAttributeClusterRevisionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientSensingUnionAttributeFeatureMapID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientSensingUnionAttributeGeneratedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientSensingUnionAttributeUnionContributorListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientSensingUnionAttributeUnionHealthID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAmbientSensingUnionAttributeUnionNameID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAppleDeviceInformationAttributeNeedsAdditionalConfigurationID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeAcceptedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeAttributeListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeBassID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeClusterRevisionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeDecreaseVolumeUnmutePolicyID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeDefaultStepSizeID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeFeatureMapID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeGeneratedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeIncreaseVolumeUnmutePolicyID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeIncreaseVolumeUnmuteVolumeID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeMaxCorrectionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeMaxDeviceVolumeDBID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeMaxDeviceVolumeID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeMaxUserVolumeID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeMidID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeMinCorrectionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeMinDeviceVolumeID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributePhysicallyMutedID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeSetVolumeUnmutePolicyID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeSoftMutedID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeStartUpMutedID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeStartUpVolumeID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeTrebleID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterAudioControlAttributeVolumeID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterBasicInformationAttributeDeviceLocationID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterBridgedDeviceBasicInformationAttributeDeviceLocationID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterCameraAVStreamManagementAttributeImageRotationDiscreteAnglesID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterCommissioningProxyAttributeAcceptedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterCommissioningProxyAttributeAttributeListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterCommissioningProxyAttributeCacheTimeoutID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterCommissioningProxyAttributeCachedResultsID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterCommissioningProxyAttributeClusterRevisionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterCommissioningProxyAttributeFeatureMapID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterCommissioningProxyAttributeGeneratedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterCommissioningProxyAttributeMaxCachedResultsID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterCommissioningProxyAttributeMaxSessionsID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterCommissioningProxyAttributeNumCachedResultsID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterCommissioningProxyAttributeScanMaxTimeID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterCommissioningProxyAttributeTransportID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterCommissioningProxyAttributeWiFiBandID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterContentLauncherAttributeMovableID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterContentLauncherAttributePresetsID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterDeviceEnergyManagementAttributePowerRangeAdjustmentID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterDynamicLightingAttributeAcceptedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterDynamicLightingAttributeAttributeListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterDynamicLightingAttributeAvailableEffectsID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterDynamicLightingAttributeClusterRevisionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterDynamicLightingAttributeCurrentEffectIDID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterDynamicLightingAttributeCurrentSpeedID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterDynamicLightingAttributeFeatureMapID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterDynamicLightingAttributeGeneratedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalAlarmAttributeAcceptedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalAlarmAttributeAttributeListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalAlarmAttributeClusterRevisionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalAlarmAttributeFeatureMapID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalAlarmAttributeGeneratedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalAlarmAttributeLatchID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalAlarmAttributeMaskID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalAlarmAttributeOverCurrentThresholdID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalAlarmAttributeOverFrequencyThresholdID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalAlarmAttributeOverPowerThresholdID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalAlarmAttributeOverVoltageThresholdID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalAlarmAttributePowerExportThresholdID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalAlarmAttributePowerImportThresholdID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalAlarmAttributeStateID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalAlarmAttributeSupportedID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalAlarmAttributeUnderCurrentThresholdID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalAlarmAttributeUnderFrequencyThresholdID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalAlarmAttributeUnderPowerThresholdID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalAlarmAttributeUnderVoltageThresholdID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalDistributionAttributeAcceptedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalDistributionAttributeAttributeListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalDistributionAttributeClusterRevisionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalDistributionAttributeEndOfLifeID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalDistributionAttributeFeatureMapID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalDistributionAttributeGeneratedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalDistributionAttributeMaxContinuousCurrentID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalDistributionAttributeMaxVoltageID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalDistributionAttributeNumberOfPolesID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalDistributionAttributeServiceEntranceRatedID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalProtectionAlarmAttributeAcceptedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalProtectionAlarmAttributeArcCauseID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalProtectionAlarmAttributeArcFaultRatingID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalProtectionAlarmAttributeAttributeListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalProtectionAlarmAttributeClusterRevisionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalProtectionAlarmAttributeFeatureMapID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalProtectionAlarmAttributeGeneratedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalProtectionAlarmAttributeMaskID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalProtectionAlarmAttributeOverLoadRatingID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalProtectionAlarmAttributeOverVoltageRatingID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalProtectionAlarmAttributeResidualCurrentRatingID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalProtectionAlarmAttributeShortCircuitRatingID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalProtectionAlarmAttributeStateID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalProtectionAlarmAttributeSupportedID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterElectricalProtectionAlarmAttributeSurgeProtectionRatingID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterGeneralDiagnosticsAttributeDeviceLoadStatusID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterGroupKeyManagementAttributeGroupcastAdoptionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterGroupcastAttributeFabricUnderTestID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterGroupcastAttributeMaxMcastAddrCountID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterGroupcastAttributeUsedMcastAddrCountID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterHumidistatAttributeAcceptedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterHumidistatAttributeAttributeListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterHumidistatAttributeClusterRevisionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterHumidistatAttributeContinuousID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterHumidistatAttributeFeatureMapID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterHumidistatAttributeGeneratedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterHumidistatAttributeMaxSetpointID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterHumidistatAttributeMinSetpointID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterHumidistatAttributeMistTypeID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterHumidistatAttributeModeID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterHumidistatAttributeOptimalID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterHumidistatAttributeSleepID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterHumidistatAttributeStepID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterHumidistatAttributeSystemStateID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterHumidistatAttributeTargetSetpointID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterHumidistatAttributeUserSetpointID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterMediaFileManagementAttributeAcceptedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterMediaFileManagementAttributeAttributeListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterMediaFileManagementAttributeAvailableFilesID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterMediaFileManagementAttributeAvailableStorageID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterMediaFileManagementAttributeClusterRevisionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterMediaFileManagementAttributeFeatureMapID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterMediaFileManagementAttributeGeneratedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterMediaFileManagementAttributeSupportedMimeTypesID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterMediaFileManagementAttributeTotalStorageID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterMediaPlaybackAttributeAvailableCommandsID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterMediaPlaybackAttributeContentInfoID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterNetworkIdentityManagementAttributeAcceptedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterNetworkIdentityManagementAttributeActiveNetworkIdentitiesID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterNetworkIdentityManagementAttributeAttributeListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterNetworkIdentityManagementAttributeClientTableSizeID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterNetworkIdentityManagementAttributeClientsID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterNetworkIdentityManagementAttributeClusterRevisionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterNetworkIdentityManagementAttributeFeatureMapID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterNetworkIdentityManagementAttributeGeneratedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterOccupancySensingAttributePredictedOccupancyID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterOperationalStateAttributeAppleOperationCompletedID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterOperationalStateAttributeAppleOperationStartTimeID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterProximityRangingAttributeAcceptedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterProximityRangingAttributeAttributeListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterProximityRangingAttributeBLEDeviceIDID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterProximityRangingAttributeBLTCSModeCapabilityID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterProximityRangingAttributeBLTCSSecurityLevelID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterProximityRangingAttributeBLTDevIKID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterProximityRangingAttributeClusterRevisionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterProximityRangingAttributeFeatureMapID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterProximityRangingAttributeGeneratedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterProximityRangingAttributeRangingCapabilitiesID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterProximityRangingAttributeSessionIDListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterProximityRangingAttributeWiFiDevIKID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterSmokeCOAlarmAttributeUnmountedID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterSmokeConcentrationMeasurementAttributeAcceptedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterSmokeConcentrationMeasurementAttributeAttributeListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterSmokeConcentrationMeasurementAttributeAverageMeasuredValueID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterSmokeConcentrationMeasurementAttributeAverageMeasuredValueWindowID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterSmokeConcentrationMeasurementAttributeClusterRevisionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterSmokeConcentrationMeasurementAttributeFeatureMapID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterSmokeConcentrationMeasurementAttributeGeneratedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterSmokeConcentrationMeasurementAttributeLevelValueID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterSmokeConcentrationMeasurementAttributeMaxMeasuredValueID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterSmokeConcentrationMeasurementAttributeMeasuredValueID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterSmokeConcentrationMeasurementAttributeMeasurementMediumID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterSmokeConcentrationMeasurementAttributeMeasurementUnitID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterSmokeConcentrationMeasurementAttributeMinMeasuredValueID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterSmokeConcentrationMeasurementAttributePeakMeasuredValueID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterSmokeConcentrationMeasurementAttributePeakMeasuredValueWindowID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterSmokeConcentrationMeasurementAttributeUncertaintyID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTemperatureControlledCabinetTopologyAttributeAcceptedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTemperatureControlledCabinetTopologyAttributeAttributeListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTemperatureControlledCabinetTopologyAttributeClusterRevisionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTemperatureControlledCabinetTopologyAttributeDisabledCabinetsID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTemperatureControlledCabinetTopologyAttributeFeatureMapID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTemperatureControlledCabinetTopologyAttributeGeneratedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTemperatureControlledCabinetTopologyAttributeTopologyID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTestHiddenManufacturerSpecificAttributeAcceptedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTestHiddenManufacturerSpecificAttributeAttributeListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTestHiddenManufacturerSpecificAttributeClusterRevisionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTestHiddenManufacturerSpecificAttributeFeatureMapID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTestHiddenManufacturerSpecificAttributeGeneratedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTestHiddenManufacturerSpecificAttributeTestAttributeID`
- `MTRBooleanStateConfigurationFeature.MTRBooleanStateConfigurationFeatureFaultEvents`
- `MTRCameraAVStreamManagementImageCodec.MTRCameraAVStreamManagementImageCodecHEIC`
- `MTRCarbonDioxideConcentrationMeasurementMeasurementUnit.MTRCarbonDioxideConcentrationMeasurementMeasurementUnitDBPM`
- `MTRCarbonDioxideConcentrationMeasurementMeasurementUnit.MTRCarbonDioxideConcentrationMeasurementMeasurementUnitPCFT`
- `MTRCarbonMonoxideConcentrationMeasurementMeasurementUnit.MTRCarbonMonoxideConcentrationMeasurementMeasurementUnitDBPM`
- `MTRCarbonMonoxideConcentrationMeasurementMeasurementUnit.MTRCarbonMonoxideConcentrationMeasurementMeasurementUnitPCFT`
- `MTRClosureControlFeature.MTRClosureControlFeatureAccess`
- `MTRClosureDimensionFeature.MTRClosureDimensionFeatureAccess`
- `MTRClusterIDType.MTRClusterIDTypeAVAnalysisID`
- `MTRClusterIDType.MTRClusterIDTypeAmbientContextSensingID`
- `MTRClusterIDType.MTRClusterIDTypeAmbientSensingUnionID`
- `MTRClusterIDType.MTRClusterIDTypeAudioControlID`
- `MTRClusterIDType.MTRClusterIDTypeCommissioningProxyID`
- `MTRClusterIDType.MTRClusterIDTypeDynamicLightingID`
- `MTRClusterIDType.MTRClusterIDTypeElectricalAlarmID`
- `MTRClusterIDType.MTRClusterIDTypeElectricalDistributionID`
- `MTRClusterIDType.MTRClusterIDTypeElectricalProtectionAlarmID`
- `MTRClusterIDType.MTRClusterIDTypeHumidistatID`
- `MTRClusterIDType.MTRClusterIDTypeMediaFileManagementID`
- `MTRClusterIDType.MTRClusterIDTypeNetworkIdentityManagementID`
- `MTRClusterIDType.MTRClusterIDTypeProximityRangingID`
- `MTRClusterIDType.MTRClusterIDTypeSmokeConcentrationMeasurementID`
- `MTRClusterIDType.MTRClusterIDTypeTemperatureControlledCabinetTopologyID`
- `MTRClusterIDType.MTRClusterIDTypeTestHiddenManufacturerSpecificID`
- `MTRCommandIDType.MTRCommandIDTypeClusterAVAnalysisCommandActivateAnalysisStreamID`
- `MTRCommandIDType.MTRCommandIDTypeClusterAVAnalysisCommandDeactivateAnalysisStreamID`
- `MTRCommandIDType.MTRCommandIDTypeClusterAVAnalysisCommandDisableContextTriggersID`
- `MTRCommandIDType.MTRCommandIDTypeClusterAVAnalysisCommandEnableContextTriggersID`
- `MTRCommandIDType.MTRCommandIDTypeClusterAVAnalysisCommandEstablishAnalysisStreamID`
- `MTRCommandIDType.MTRCommandIDTypeClusterAVAnalysisCommandEstablishAnalysisStreamResponseID`
- `MTRCommandIDType.MTRCommandIDTypeClusterAVAnalysisCommandRemoveAnalysisStreamID`
- `MTRCommandIDType.MTRCommandIDTypeClusterAudioControlCommandDecreaseVolumeID`
- `MTRCommandIDType.MTRCommandIDTypeClusterAudioControlCommandIncreaseVolumeID`
- `MTRCommandIDType.MTRCommandIDTypeClusterAudioControlCommandMuteID`
- `MTRCommandIDType.MTRCommandIDTypeClusterAudioControlCommandSetVolumeID`
- `MTRCommandIDType.MTRCommandIDTypeClusterAudioControlCommandToggleMutedID`
- `MTRCommandIDType.MTRCommandIDTypeClusterAudioControlCommandUnmuteID`
- `MTRCommandIDType.MTRCommandIDTypeClusterClosureControlCommandGroupedMoveToID`
- `MTRCommandIDType.MTRCommandIDTypeClusterClosureDimensionCommandGroupedSetTargetID`
- `MTRCommandIDType.MTRCommandIDTypeClusterClosureDimensionCommandGroupedStepID`
- `MTRCommandIDType.MTRCommandIDTypeClusterCommissioningProxyCommandProxyBackGroundScanStartRequestID`
- `MTRCommandIDType.MTRCommandIDTypeClusterCommissioningProxyCommandProxyBackGroundScanStopRequestID`
- `MTRCommandIDType.MTRCommandIDTypeClusterCommissioningProxyCommandProxyConnectRequestID`
- `MTRCommandIDType.MTRCommandIDTypeClusterCommissioningProxyCommandProxyConnectResponseID`
- `MTRCommandIDType.MTRCommandIDTypeClusterCommissioningProxyCommandProxyDisconnectRequestID`
- `MTRCommandIDType.MTRCommandIDTypeClusterCommissioningProxyCommandProxyMessageRequestID`
- `MTRCommandIDType.MTRCommandIDTypeClusterCommissioningProxyCommandProxyMessageResponseID`
- `MTRCommandIDType.MTRCommandIDTypeClusterCommissioningProxyCommandProxyScanRequestID`
- `MTRCommandIDType.MTRCommandIDTypeClusterCommissioningProxyCommandProxyScanResponseID`
- `MTRCommandIDType.MTRCommandIDTypeClusterContentLauncherCommandContentReplicationRequestID`
- `MTRCommandIDType.MTRCommandIDTypeClusterContentLauncherCommandContentReplicationResponseID`
- `MTRCommandIDType.MTRCommandIDTypeClusterContentLauncherCommandPlayPresetID`
- `MTRCommandIDType.MTRCommandIDTypeClusterDeviceEnergyManagementCommandCancelPowerRangeAdjustRequestID`
- `MTRCommandIDType.MTRCommandIDTypeClusterDeviceEnergyManagementCommandPowerRangeAdjustRequestID`
- `MTRCommandIDType.MTRCommandIDTypeClusterDynamicLightingCommandStartEffectID`
- `MTRCommandIDType.MTRCommandIDTypeClusterDynamicLightingCommandStopEffectID`
- `MTRCommandIDType.MTRCommandIDTypeClusterElectricalAlarmCommandModifyEnabledAlarmsID`
- `MTRCommandIDType.MTRCommandIDTypeClusterElectricalAlarmCommandResetID`
- `MTRCommandIDType.MTRCommandIDTypeClusterElectricalAlarmCommandSetElectricalAlarmThresholdsID`
- `MTRCommandIDType.MTRCommandIDTypeClusterElectricalProtectionAlarmCommandModifyEnabledAlarmsID`
- `MTRCommandIDType.MTRCommandIDTypeClusterGroupcastCommandGroupcastTestingID`
- `MTRCommandIDType.MTRCommandIDTypeClusterHumidistatCommandSetSettingsID`
- `MTRCommandIDType.MTRCommandIDTypeClusterMediaFileManagementCommandAddFileID`
- `MTRCommandIDType.MTRCommandIDTypeClusterMediaFileManagementCommandAddFileResponseID`
- `MTRCommandIDType.MTRCommandIDTypeClusterMediaFileManagementCommandDeleteFileID`
- `MTRCommandIDType.MTRCommandIDTypeClusterMediaFileManagementCommandGetSharedFileID`
- `MTRCommandIDType.MTRCommandIDTypeClusterMediaFileManagementCommandGetSharedFileResponseID`
- `MTRCommandIDType.MTRCommandIDTypeClusterMediaFileManagementCommandOfferFileID`
- `MTRCommandIDType.MTRCommandIDTypeClusterMediaFileManagementCommandRequestSharedFilesID`
- `MTRCommandIDType.MTRCommandIDTypeClusterNetworkIdentityManagementCommandAddClientID`
- `MTRCommandIDType.MTRCommandIDTypeClusterNetworkIdentityManagementCommandAddClientResponseID`
- `MTRCommandIDType.MTRCommandIDTypeClusterNetworkIdentityManagementCommandExportAdminSecretID`
- `MTRCommandIDType.MTRCommandIDTypeClusterNetworkIdentityManagementCommandExportAdminSecretResponseID`
- `MTRCommandIDType.MTRCommandIDTypeClusterNetworkIdentityManagementCommandImportAdminSecretID`
- `MTRCommandIDType.MTRCommandIDTypeClusterNetworkIdentityManagementCommandQueryIdentityID`
- `MTRCommandIDType.MTRCommandIDTypeClusterNetworkIdentityManagementCommandQueryIdentityResponseID`
- `MTRCommandIDType.MTRCommandIDTypeClusterNetworkIdentityManagementCommandRemoveClientID`
- `MTRCommandIDType.MTRCommandIDTypeClusterProximityRangingCommandStartRangingRequestID`
- `MTRCommandIDType.MTRCommandIDTypeClusterProximityRangingCommandStartRangingResponseID`
- `MTRCommandIDType.MTRCommandIDTypeClusterProximityRangingCommandStopRangingRequestID`
- `MTRContentLauncherFeature.MTRContentLauncherFeatureContentQueueing`
- `MTRContentLauncherFeature.MTRContentLauncherFeatureContentReplication`
- `MTRContentLauncherFeature.MTRContentLauncherFeaturePresets`
- `MTRContentLauncherStatus.MTRContentLauncherStatusAccountMismatch`
- `MTRContentLauncherStatus.MTRContentLauncherStatusContentAppNotAvailable`
- `MTRContentLauncherStatus.MTRContentLauncherStatusInvalidData`
- `MTRContentLauncherStatus.MTRContentLauncherStatusPresetNotFound`
- `MTRContentLauncherStatus.MTRContentLauncherStatusReplicationNotAllowed`
- `MTRContentLauncherStatus.MTRContentLauncherStatusReplicationNotSupported`
- `MTRDeviceEnergyManagementCause.MTRDeviceEnergyManagementCauseInvalid`
- `MTRDeviceEnergyManagementFeature.MTRDeviceEnergyManagementFeaturePowerRangeAdjustment`
- `MTRDeviceTypeIDType.MTRDeviceTypeIDTypeAmbientContextSensorID`
- `MTRDeviceTypeIDType.MTRDeviceTypeIDTypeCommissioningByProxyID`
- `MTRDeviceTypeIDType.MTRDeviceTypeIDTypeElectricalCircuitBreakerID`
- `MTRDeviceTypeIDType.MTRDeviceTypeIDTypeElectricalDistributionEnclosureID`
- `MTRDeviceTypeIDType.MTRDeviceTypeIDTypeHumidifierDehumidifierID`
- `MTRDeviceTypeIDType.MTRDeviceTypeIDTypeProximityRangerID`
- `MTREventIDType.MTREventIDTypeClusterAVAnalysisEventAnalysisSessionEndID`
- `MTREventIDType.MTREventIDTypeClusterAVAnalysisEventAnalysisSessionStartID`
- `MTREventIDType.MTREventIDTypeClusterAVAnalysisEventPerceivedContextID`
- `MTREventIDType.MTREventIDTypeClusterAmbientContextSensingEventAmbientContextDetectEndedID`
- `MTREventIDType.MTREventIDTypeClusterAmbientContextSensingEventAmbientContextDetectStartedID`
- `MTREventIDType.MTREventIDTypeClusterAmbientSensingUnionEventUnionContributorAddedID`
- `MTREventIDType.MTREventIDTypeClusterAmbientSensingUnionEventUnionContributorRemovedID`
- `MTREventIDType.MTREventIDTypeClusterAmbientSensingUnionEventUnionContributorStatusChangedID`
- `MTREventIDType.MTREventIDTypeClusterChimeEventChimeStartedPlayingID`
- `MTREventIDType.MTREventIDTypeClusterContentLauncherEventContentReplicationID`
- `MTREventIDType.MTREventIDTypeClusterDeviceEnergyManagementEventPowerRangeAdjustEndID`
- `MTREventIDType.MTREventIDTypeClusterDeviceEnergyManagementEventPowerRangeAdjustStartID`
- `MTREventIDType.MTREventIDTypeClusterElectricalAlarmEventNotifyID`
- `MTREventIDType.MTREventIDTypeClusterElectricalProtectionAlarmEventNotifyID`
- `MTREventIDType.MTREventIDTypeClusterGroupcastEventGroupcastTestingID`
- `MTREventIDType.MTREventIDTypeClusterMediaFileManagementEventSharedFilesAddedID`
- `MTREventIDType.MTREventIDTypeClusterProximityRangingEventRangingResultID`
- `MTREventIDType.MTREventIDTypeClusterProximityRangingEventRangingSessionStatusID`
- `MTRFormaldehydeConcentrationMeasurementMeasurementUnit.MTRFormaldehydeConcentrationMeasurementMeasurementUnitDBPM`
- `MTRFormaldehydeConcentrationMeasurementMeasurementUnit.MTRFormaldehydeConcentrationMeasurementMeasurementUnitPCFT`
- `MTRGroupKeyManagementFeature.MTRGroupKeyManagementFeatureGroupcast`
- `MTRGroupcastFeature.MTRGroupcastFeaturePerGroup`
- `MTRNitrogenDioxideConcentrationMeasurementMeasurementUnit.MTRNitrogenDioxideConcentrationMeasurementMeasurementUnitDBPM`
- `MTRNitrogenDioxideConcentrationMeasurementMeasurementUnit.MTRNitrogenDioxideConcentrationMeasurementMeasurementUnitPCFT`
- `MTROccupancySensingFeature.MTROccupancySensingFeatureOccupancyEvent`
- `MTROccupancySensingFeature.MTROccupancySensingFeaturePrediction`
- `MTROvenModeModeTag.MTROvenModeModeTagAirFry`
- `MTROvenModeModeTag.MTROvenModeModeTagAirSousVide`
- `MTROvenModeModeTag.MTROvenModeModeTagFrozenFood`
- `MTROvenModeModeTag.MTROvenModeModeTagSteam`
- `MTROzoneConcentrationMeasurementMeasurementUnit.MTROzoneConcentrationMeasurementMeasurementUnitDBPM`
- `MTROzoneConcentrationMeasurementMeasurementUnit.MTROzoneConcentrationMeasurementMeasurementUnitPCFT`
- `MTRPM10ConcentrationMeasurementMeasurementUnit.MTRPM10ConcentrationMeasurementMeasurementUnitDBPM`
- `MTRPM10ConcentrationMeasurementMeasurementUnit.MTRPM10ConcentrationMeasurementMeasurementUnitPCFT`
- `MTRPM1ConcentrationMeasurementMeasurementUnit.MTRPM1ConcentrationMeasurementMeasurementUnitDBPM`
- `MTRPM1ConcentrationMeasurementMeasurementUnit.MTRPM1ConcentrationMeasurementMeasurementUnitPCFT`
- `MTRPM25ConcentrationMeasurementMeasurementUnit.MTRPM25ConcentrationMeasurementMeasurementUnitDBPM`
- `MTRPM25ConcentrationMeasurementMeasurementUnit.MTRPM25ConcentrationMeasurementMeasurementUnitPCFT`
- `MTRPushAVStreamTransportStatusCode.MTRPushAVStreamTransportStatusCodeDuplicateStreamValues`
- `MTRPushAVStreamTransportStatusCode.MTRPushAVStreamTransportStatusCodeInvalidPreRollLength`
- `MTRPushAVStreamTransportTransportTriggerType.MTRPushAVStreamTransportTransportTriggerTypeAmbientContext`
- `MTRPushAVStreamTransportTriggerActivationReason.MTRPushAVStreamTransportTriggerActivationReasonDoorbellPressed`
- `MTRRadonConcentrationMeasurementMeasurementUnit.MTRRadonConcentrationMeasurementMeasurementUnitDBPM`
- `MTRRadonConcentrationMeasurementMeasurementUnit.MTRRadonConcentrationMeasurementMeasurementUnitPCFT`
- `MTRSmokeCOAlarmExpressedState.MTRSmokeCOAlarmExpressedStateInoperative`
- `MTRTotalVolatileOrganicCompoundsConcentrationMeasurementMeasurementUnit.MTRTotalVolatileOrganicCompoundsConcentrationMeasurementMeasurementUnitDBPM`
- `MTRTotalVolatileOrganicCompoundsConcentrationMeasurementMeasurementUnit.MTRTotalVolatileOrganicCompoundsConcentrationMeasurementMeasurementUnitPCFT`
- `MTRTransportType.MTRTransportTypeNFC`
- `MTRTransportType.MTRTransportTypeThreadMeshcop`
- `MTRTransportType.MTRTransportTypeWiFiPAF`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Enum members removed (15)

- `MTRAttributeIDType.MTRAttributeIDTypeClusterTimerAttributeAcceptedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTimerAttributeAttributeListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTimerAttributeClusterRevisionID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTimerAttributeFeatureMapID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTimerAttributeGeneratedCommandListID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTimerAttributeSetTimeID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTimerAttributeTimeRemainingID`
- `MTRAttributeIDType.MTRAttributeIDTypeClusterTimerAttributeTimerStateID`
- `MTRClusterIDType.MTRClusterIDTypeTimerID`
- `MTRCommandIDType.MTRCommandIDTypeClusterGroupcastCommandExpireGracePeriodID`
- `MTRCommandIDType.MTRCommandIDTypeClusterTimerCommandAddTimeID`
- `MTRCommandIDType.MTRCommandIDTypeClusterTimerCommandReduceTimeID`
- `MTRCommandIDType.MTRCommandIDTypeClusterTimerCommandResetTimerID`
- `MTRCommandIDType.MTRCommandIDTypeClusterTimerCommandSetTimerID`
- `MTRJointFabricDatastoreDatastoreAccessControlEntryPrivilege.MTRJointFabricDatastoreDatastoreAccessControlEntryPrivilegeProxyView`

### Externs added (2)

- `MTRCommissioningSessionTransportType`
- `MTRUnpoweredInitialPhase`

## MatterSupport

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## MediaAccessibility

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (6)

- `MACaptionAppearanceDomain.kMACaptionAppearanceDomainVideoConferencing`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Function signature changes (2)

- `MAImageCaptioningCopyCaption`
  - old: `(CFURLRef _Nonnull, CF_RETURNS_RETAINED CFErrorRef  _Nullable *) → CFStringRef  _Nullable`
  - new: `(CFURLRef _Nonnull, CFErrorRef  _Nullable *) → CFStringRef  _Nullable`
- `MAImageCaptioningSetCaption`
  - old: `(CFURLRef _Nonnull, CFStringRef _Nullable, CF_RETURNS_RETAINED CFErrorRef  _Nullable *) → bool`
  - new: `(CFURLRef _Nonnull, CFStringRef _Nullable, CFErrorRef  _Nullable *) → bool`

## MediaExtension

### Methods added (2)

- `MEFileInfo -constituentFileNames`
- `MEFileInfo -setConstituentFileNames:`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## MediaLibrary

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## MediaPlayer

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Externs added (1)

- `MPNowPlayingInfoPropertyAppEntityIdentifiers`

## MediaToolbox

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Functions added (1)

- `MTAudioProcessingTapCreateWithPreferredFormat`

## Metal

### Classes added (4)

- `MTLTensorAuxiliaryPlaneDescriptor`
- `MTLTensorAuxiliaryPlaneDescriptorMap`
- `MTLTensorAuxiliaryPlaneType`
- `MTLTensorBufferAttachments`

### Methods added (19)

- `MTL4ComputePipelineDescriptor -contentionRelief`
- `MTL4ComputePipelineDescriptor -forwardProgressUsage`
- `MTL4ComputePipelineDescriptor -optimizeForPersistentKernel`
- `MTL4ComputePipelineDescriptor -setContentionRelief:`
- `MTL4ComputePipelineDescriptor -setForwardProgressUsage:`
- `MTL4ComputePipelineDescriptor -setOptimizeForPersistentKernel:`
- `MTLCompileOptions -floatingPointConversionRoundingMode`
- `MTLCompileOptions -setFloatingPointConversionRoundingMode:`
- `MTLComputePipelineDescriptor -contentionRelief`
- `MTLComputePipelineDescriptor -forwardProgressUsage`
- `MTLComputePipelineDescriptor -optimizeForPersistentKernel`
- `MTLComputePipelineDescriptor -setContentionRelief:`
- `MTLComputePipelineDescriptor -setForwardProgressUsage:`
- `MTLComputePipelineDescriptor -setOptimizeForPersistentKernel:`
- `MTLTensorDescriptor -auxiliaryPlanes`
- `MTLTensorDescriptor -setAuxiliaryPlanes:`
- `MTLTensorReferenceType -auxiliaryPlanes`
- `MTLTextureViewDescriptor -minLOD`
- `MTLTextureViewDescriptor -setMinLOD:`

### Protocols added (1)

- `MTLTensorAuxiliaryPlane`

### Enums added (5)

- `MTLContentionRelief`
- `MTLFloatingPointConversionRoundingMode`
- `MTLForwardProgressUsage`
- `MTLTensorPlaneType`
- `task_shared_region_stubs_t`

### Enum members added (25)

- `MTLGPUFamily.MTLGPUFamilyApple11`
- `MTLLanguageVersion.MTLLanguageVersion4_1`
- `MTLPixelFormat.MTLPixelFormatRGB16Float`
- `MTLPixelFormat.MTLPixelFormatRGB16Sint`
- `MTLPixelFormat.MTLPixelFormatRGB16Snorm`
- `MTLPixelFormat.MTLPixelFormatRGB16Uint`
- `MTLPixelFormat.MTLPixelFormatRGB16Unorm`
- `MTLPixelFormat.MTLPixelFormatRGB32Float`
- `MTLPixelFormat.MTLPixelFormatRGB32Sint`
- `MTLPixelFormat.MTLPixelFormatRGB32Uint`
- `MTLPixelFormat.MTLPixelFormatRGB8Sint`
- `MTLPixelFormat.MTLPixelFormatRGB8Snorm`
- `MTLPixelFormat.MTLPixelFormatRGB8Uint`
- `MTLPixelFormat.MTLPixelFormatRGB8Unorm`
- `MTLTensorDataType.MTLTensorDataTypeInt2`
- `MTLTensorDataType.MTLTensorDataTypeMetalFloat4E2M1`
- `MTLTensorDataType.MTLTensorDataTypeMetalFloat8E4M3`
- `MTLTensorDataType.MTLTensorDataTypeMetalFloat8E5M2`
- `MTLTensorDataType.MTLTensorDataTypeMetalFloat8UE8M0`
- `MTLTensorDataType.MTLTensorDataTypeUInt2`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Function signature changes (3)

- `MTLCopyAllDevices`
  - old: `() → NS_RETURNS_RETAINED NSArray<id<MTLDevice>> * _Nonnull`
  - new: `() → NSArray<id<MTLDevice>> * _Nonnull`
- `MTLCopyAllDevicesWithObserver`
  - old: `(id<NSObject>  _Nullable * _Nonnull, MTLDeviceNotificationHandler _Nonnull) → NS_RETURNS_RETAINED NSArray<id<MTLDevice>> * _Nonnull`
  - new: `(id<NSObject>  _Nullable * _Nonnull, MTLDeviceNotificationHandler _Nonnull) → NSArray<id<MTLDevice>> * _Nonnull`
- `MTLCreateSystemDefaultDevice`
  - old: `() → NS_RETURNS_RETAINED id<MTLDevice>  _Nullable`
  - new: `() → id<MTLDevice>  _Nullable`

## MetalFX

### Methods added (8)

- `MTLFXFrameInterpolatorDescriptor -isDistortionTextureEnabled`
- `MTLFXFrameInterpolatorDescriptor -requiresPrevColorTexture`
- `MTLFXFrameInterpolatorDescriptor -setDistortionTextureEnabled:`
- `MTLFXFrameInterpolatorDescriptor -setRequiresPrevColorTexture:`
- `MTLFXTemporalScalerDescriptor -isJitteredMotionVectorsEnabled`
- `MTLFXTemporalScalerDescriptor -isOutputResolutionMotionVectorsEnabled`
- `MTLFXTemporalScalerDescriptor -setJitteredMotionVectorsEnabled:`
- `MTLFXTemporalScalerDescriptor -setOutputResolutionMotionVectorsEnabled:`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## MetalKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## MetalPerformanceShaders

### Classes added (2)

- `MPSFColorConversion`
- `MPSFunction`

### Methods added (5)

- `MPSNDArrayIdentity -reshapeWithMTL4CommandEncoder:sourceArray:dimensionCount:dimensionSizes:destinationArray:`
- `MPSNDArrayIdentity -reshapeWithMTL4CommandEncoder:sourceArray:shape:destinationArray:`
- `MPSNDArrayIdentity -reshapeWithSourceArray:shape:`
- `MPSNDArrayMultiaryKernel -encodeWithMTL4CommandEncoder:sourceArrays:destinationArray:`
- `MPSNDArrayUnaryKernel -encodeWithMTL4CommandEncoder:sourceArray:destinationArray:`

### Enums added (3)

- `EvCmd`
- `NXMouseButton`
- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## MetalPerformanceShadersGraph

### Classes added (1)

- `MPSGraphSDPADescriptor`

### Methods added (6)

- `MPSGraph -runAsyncWithMTL4CommandQueue:feeds:targetOperations:resultsDictionary:executionDescriptor:`
- `MPSGraph -runAsyncWithMTL4CommandQueue:feeds:targetTensors:targetOperations:executionDescriptor:`
- `MPSGraph -scaledDotProductAttentionWithQueryTensor:keyTensor:valueTensor:descriptor:name:`
- `MPSGraphCompilationDescriptor -disableAutoLayoutConversion`
- `MPSGraphExecutable -runAsyncWithMTL4CommandQueue:inputsArray:resultsArray:executionDescriptor:`
- `MPSGraphExecutable -runWithMTL4CommandQueue:inputsArray:resultsArray:executionDescriptor:`

### Enums added (3)

- `EvCmd`
- `NXMouseButton`
- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## MetricKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Deprecation changes (36)

- `MXAnimationMetric: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXAppExitMetric: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXAppLaunchMetric: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXAppResponsivenessMetric: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXAppRunTimeMetric: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXAverage: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXBackgroundExitData: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXCPUExceptionDiagnostic: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXCPUMetric: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXCallStackTree: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXCellularConditionMetric: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXCrashDiagnostic: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXCrashDiagnosticObjectiveCExceptionReason: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXDiagnostic: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXDiagnosticPayload: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXDiskIOMetric: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXDiskSpaceUsageMetric: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXDiskWriteExceptionDiagnostic: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXDisplayMetric: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXForegroundExitData: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXGPUMetric: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXHangDiagnostic: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXHistogram: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXHistogramBucket: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXLocationActivityMetric: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXMemoryMetric: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXMetaData: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXMetric: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXMetricManager: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXMetricPayload: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXNetworkTransferMetric: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXSignpostIntervalData: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXSignpostMetric: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXSignpostRecord: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXUnitAveragePixelLuminance: deprecated (none) → API_TO_BE_DEPRECATED`
- `MXUnitSignalBars: deprecated (none) → API_TO_BE_DEPRECATED`

## ModelIO

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## MultipeerConnectivity

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Deprecation changes (6)

- `MCAdvertiserAssistant: deprecated (none) → 27.0`
- `MCBrowserViewController: deprecated (none) → 27.0`
- `MCNearbyServiceAdvertiser: deprecated (none) → 27.0`
- `MCNearbyServiceBrowser: deprecated (none) → 27.0`
- `MCPeerID: deprecated (none) → 27.0`
- `MCSession: deprecated (none) → 27.0`

## NaturalLanguage

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## NearbyInteraction

### Classes added (1)

- `NIDLTDOAMeasurementFloorElevation`

### Methods added (10)

- `NIDLTDOAConfiguration -discoveryMethod`
- `NIDLTDOAConfiguration -initWithNetworkIdentifier:discoveryMethod:`
- `NIDLTDOAConfiguration -setDiscoveryMethod:`
- `NIDLTDOAMeasurement -clusterInitiatorAddress`
- `NIDLTDOAMeasurement -floorElevation`
- `NIDLTDOAMeasurement -rawReceiveTime`
- `NIDLTDOAMeasurement -rawTransmitTime`
- `NIDLTDOAMeasurement -responderClockFrequencyOffset`
- `NINearbyAccessoryConfiguration -initWithBluetoothChannelSoundingIdentifier:previousBluetoothIdentifier:`
- `NISession -updateMotionState:forObjectWithToken:`

### Method signature changes (2)

- `NIDLTDOAMeasurement +new`
  - old: `() → instancetype`
  - new: `() → instancetype _Nonnull`
- `NIDLTDOAMeasurement -init`
  - old: `() → instancetype`
  - new: `() → instancetype _Nonnull`

### Enums added (3)

- `NIDLTDOADiscoveryMethod`
- `NIMotionActivityState`
- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## Network

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Enum base type changes (1)

- `nw_link_quality_t`
  - old: `int32`
  - new: `uint8`

### Functions added (1)

- `nw_tcp_set_max_pacing_rate`

## NetworkExtension

### Methods added (14)

- `NEAppPushManager -matchMissionCriticalService`
- `NEAppPushManager -setMatchMissionCriticalService:`
- `NEPacketTunnelNetworkSettings -enforceRoutes`
- `NEPacketTunnelNetworkSettings -excludeAPNs`
- `NEPacketTunnelNetworkSettings -excludeCellularServices`
- `NEPacketTunnelNetworkSettings -excludeDeviceCommunication`
- `NEPacketTunnelNetworkSettings -excludeLocalNetworks`
- `NEPacketTunnelNetworkSettings -includeAllNetworks`
- `NEPacketTunnelNetworkSettings -setEnforceRoutes:`
- `NEPacketTunnelNetworkSettings -setExcludeAPNs:`
- `NEPacketTunnelNetworkSettings -setExcludeCellularServices:`
- `NEPacketTunnelNetworkSettings -setExcludeDeviceCommunication:`
- `NEPacketTunnelNetworkSettings -setExcludeLocalNetworks:`
- `NEPacketTunnelNetworkSettings -setIncludeAllNetworks:`

### Method signature changes (2)

- `NENetworkRule -matchLocalNetwork`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT NWHostEndpoint *`
  - new: `() → NWHostEndpoint *`
- `NENetworkRule -matchRemoteEndpoint`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT NWHostEndpoint *`
  - new: `() → NWHostEndpoint *`

### Enums added (2)

- `NEPacketTunnelNetworkSettingsIPFamily`
- `task_shared_region_stubs_t`

### Enum members added (6)

- `NEVPNIKEv2TLSVersion.NEVPNIKEv2TLSVersion1_3`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Enum base type changes (1)

- `nw_link_quality_t`
  - old: `int32`
  - new: `uint8`

## NotificationCenter

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## OSAKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## OSLog

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## OSServices

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## OpenDirectory

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## OpenScripting

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## PDFKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## PHASE

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ParavirtualizedGraphics

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Function signature changes (1)

- `PGCreateDeviceWithDescriptor`
  - old: `(PGDeviceDescriptor * _Nonnull) → NS_RETURNS_RETAINED id<PGDevice>  _Nullable`
  - new: `(PGDeviceDescriptor * _Nonnull) → id<PGDevice>  _Nullable`

### Externs removed (2)

- `ParavirtualizedGraphicsVersionNumber`
- `ParavirtualizedGraphicsVersionString`

## PassKit

### Methods added (9)

- `PKDisbursementRequest -setUnsupportedPrimaryAccountIdentifiers:`
- `PKDisbursementRequest -unsupportedPrimaryAccountIdentifiers`
- `PKIdentityElement +nameElement`
- `PKPassLibrary -addPassesFromArchiveAtFileURL:completionHandler:`
- `PKPassLibrary -addPassesFromArchiveWithData:completionHandler:`
- `PKPassLibrary -addPassesWithData:completionHandler:`
- `PKPaymentRequest -setUnsupportedPrimaryAccountIdentifiers:`
- `PKPaymentRequest -unsupportedPrimaryAccountIdentifiers`
- `PKSecureElementPass -isProvisioningAvailable`

### Method signature changes (1)

- `PKShareablePassMetadata -templateIdentifier`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT NSString *`
  - new: `() → NSString *`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Externs added (7)

- `PKPaymentNetworkElcard`
- `PKPaymentNetworkHumo`
- `PKPaymentNetworkMaal`
- `PKPaymentNetworkPayPak`
- `PKPaymentNetworkRuPay`
- `PKPaymentNetworkUzCard`
- `PKPaymentNetworkVerve`

## PencilKit

### Classes added (2)

- `PKConvertedBezierPoint`
- `PKStrokeRenderState`

### Methods added (13)

- `PKDrawing -drawingByErasingStrokePath:mask:transform:`
- `PKStroke -initWithInk:strokePath:transform:mask:randomSeed:strokeID:renderGroupID:renderState:`
- `PKStroke -renderGroupID`
- `PKStroke -renderState`
- `PKStroke -strokeID`
- `PKStroke -substrokeWithRange:`
- `PKStrokePath -bezierRepresentation`
- `PKStrokePath -initWithBezierPath:creationDate:pointProvider:`
- `PKStrokePath -initWithControlPoints:creationDate:strokePathID:`
- `PKStrokePath -strokePathID`
- `PKStrokePath -subpathWithRange:`
- `PKStrokePoint -initWithLocation:timeOffset:size:opacity:force:azimuth:altitude:secondaryScale:threshold:lateralJitter:`
- `PKStrokePoint -lateralJitter`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (6)

- `PKContentVersion.PKContentVersion5`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## Photos

### Classes added (2)

- `PHAssetExtendedMetadata`
- `PHAssetResourceUploadJobOptions`

### Methods added (43)

- `PHAsset -adjustmentTimestamp`
- `PHAsset -adjustmentsState`
- `PHAsset -extendedMetadata`
- `PHAsset -originalResourceChoice`
- `PHAsset -playbackVariation`
- `PHAsset -rating`
- `PHAssetChangeRequest -addKeyword:`
- `PHAssetChangeRequest -caption`
- `PHAssetChangeRequest -rating`
- `PHAssetChangeRequest -removeKeyword:`
- `PHAssetChangeRequest -revertAssetContentToOriginalResourceChoice:`
- `PHAssetChangeRequest -setCaption:`
- `PHAssetChangeRequest -setLivePhotoVideoPlaybackEnabled:`
- `PHAssetChangeRequest -setRating:`
- `PHAssetCreationRequest -originalResourceChoice`
- `PHAssetCreationRequest -setOriginalResourceChoice:`
- `PHAssetResource +assetResourceForUploadJob:`
- `PHAssetResource -dataSize`
- `PHAssetResource -filename`
- `PHCloudIdentifier -archivalStringValue`
- `PHCloudIdentifier -initWithArchivalStringValue:`
- `PHCollection -modificationDate`
- `PHContentEditingInputRequestOptions -originalResourceChoice`
- `PHContentEditingInputRequestOptions -setOriginalResourceChoice:`
- `PHContentEditingInputRequestOptions -setSkipsDisplaySizeImage:`
- `PHContentEditingInputRequestOptions -skipsDisplaySizeImage`
- `PHFetchOptions -prefetchAssetExtendedMetadata`
- `PHFetchOptions -setPrefetchAssetExtendedMetadata:`
- `PHImageRequestOptions -preferHDR`
- `PHImageRequestOptions -setPreferHDR:`
- `PHImageRequestOptions -setTargetHDRHeadroom:`
- `PHImageRequestOptions -targetHDRHeadroom`
- `PHLivePhotoRequestOptions -preferHDR`
- `PHLivePhotoRequestOptions -setPreferHDR:`
- `PHPhotoLibrary -disableUploadJobExtensionWithError:`
- `PHPhotoLibrary -enableUploadJobExtensionWithOptions:error:`
- `PHPhotoLibrary -isUploadJobExtensionEnabled`
- `PHPhotoLibrary -localIdentifierMappingsForSyncedCloudIdentifiers:`
- `PHPhotoLibrary -registerPersistentChangesObserver:`
- `PHPhotoLibrary -setUploadJobExtensionEnabled:error:`
- `PHPhotoLibrary -setUploadJobExtensionOptions:error:`
- `PHPhotoLibrary -unregisterPersistentChangesObserver:`
- `PHPhotoLibrary -uploadJobExtensionOptions`

### Method signature changes (4)

- `PHAsset -addedDate`
  - old: `() → NSDate * _Nonnull`
  - new: `() → NSDate * _Nullable`
- `PHAssetResourceUploadJob -resource`
  - old: `() → PHAssetResource * _Nonnull`
  - new: `() → PHAssetResource *`
- `PHCloudIdentifier +notFoundIdentifier`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT PHCloudIdentifier *`
  - new: `() → PHCloudIdentifier *`
- `PHContentEditingInput -avAsset`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT AVAsset *`
  - new: `() → AVAsset *`

### Protocols added (1)

- `PHPhotoLibraryPersistentChangesObserver`

### Enums added (5)

- `PHAssetAdjustmentsState`
- `PHAssetPlaybackVariation`
- `PHAssetRating`
- `PHOriginalResourceChoice`
- `task_shared_region_stubs_t`

### Enum members added (7)

- `PHAssetMediaSubtype.PHAssetMediaSubtypePhotoAnimation`
- `PHCollectionListSubtype.PHCollectionListSubtypeRootFolder`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## PhotosUI

### Classes added (6)

- `PHPickerSearchText`
- `PHSharedAlbumCreationConfiguration`
- `PHSharedAlbumCreationResult`
- `PHSharedAlbumCreationViewController`
- `PHSharedAlbumCustomizationViewController`
- `PHSharedAlbumPostingViewController`

### Methods added (6)

- `PHPickerConfiguration -metadataOptions`
- `PHPickerConfiguration -searchText`
- `PHPickerConfiguration -setMetadataOptions:`
- `PHPickerConfiguration -setSearchText:`
- `PHPickerUpdateConfiguration -searchText`
- `PHPickerUpdateConfiguration -setSearchText:`

### Methods removed (3)

- `PHPickerFilter +new`
- `PHPickerResult +new`
- `PHPickerViewController +new`

### Protocols added (3)

- `PHSharedAlbumCreationViewControllerDelegate`
- `PHSharedAlbumCustomizationViewControllerDelegate`
- `PHSharedAlbumPostingViewControllerDelegate`

### Enums added (3)

- `PHPickerMetadataOptions`
- `PHSharedAlbumCreationSharingPolicy`
- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## PowerSources

### Enum base type changes (12)

- `IOPSLowBatteryWarningLevel`
  - old: `int64`
  - new: `int32`
- `acl_entry_id_t`
  - old: `int64`
  - new: `int32`
- `acl_flag_t`
  - old: `int64`
  - new: `int32`
- `acl_perm_t`
  - old: `int64`
  - new: `int32`
- `acl_tag_t`
  - old: `int64`
  - new: `int32`
- `acl_type_t`
  - old: `int64`
  - new: `int32`
- `clockid_t`
  - old: `int64`
  - new: `int32`
- `filesec_property_t`
  - old: `int64`
  - new: `int32`
- `idtype_t`
  - old: `int64`
  - new: `int32`
- `mpo_flags_t`
  - old: `int64`
  - new: `uint32`
- `os_clockid_t`
  - old: `int64`
  - new: `uint32`
- `ptrauth_key`
  - old: `int64`
  - new: `int32`

## PreferencePanes

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## PrintCore

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ProximityReaderStub

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## PushKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## QD

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## Quartz

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## QuartzComposer

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## QuartzCore

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## QuartzFilters

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## QuickLook

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## QuickLookThumbnailing

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## QuickLookUI

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## RelevanceKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ReplayKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Deprecation changes (6)

- `RPBroadcastActivityController: deprecated (none) → 27.0`
- `RPBroadcastController: deprecated (none) → 27.0`
- `RPBroadcastHandler: deprecated (none) → 27.0`
- `RPBroadcastSampleHandler: deprecated (none) → 27.0`
- `RPPreviewViewController: deprecated (none) → 27.0`
- `RPScreenRecorder: deprecated (none) → 27.0`

## SafariServices

### Classes added (1)

- `SFSafariSettings`

### Enums added (2)

- `SFSafariSettingsError`
- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Externs added (1)

- `SFSafariSettingsErrorDomain`

## SafetyKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## SceneKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ScreenCaptureKit

### Classes added (3)

- `SCClipBufferingOutput`
- `SCRecordingEditor`
- `SCVideoEffectOutput`

### Methods added (15)

- `SCContentFilter -isCameraEnabled`
- `SCContentFilter -isMicrophoneEnabled`
- `SCContentSharingPicker -isAvailable`
- `SCContentSharingPicker -presentPickerForCurrentApplication`
- `SCContentSharingPickerConfiguration -setShowsCameraControl:`
- `SCContentSharingPickerConfiguration -setShowsMicrophoneControl:`
- `SCContentSharingPickerConfiguration -showsCameraControl`
- `SCContentSharingPickerConfiguration -showsMicrophoneControl`
- `SCRecordingOutputConfiguration -mixesAudioWithMicrophone`
- `SCRecordingOutputConfiguration -setMixesAudioWithMicrophone:`
- `SCStream -addClipBufferingOutput:error:`
- `SCStream -addVideoEffectOutput:error:`
- `SCStream -isCapturing`
- `SCStream -removeClipBufferingOutput:error:`
- `SCStream -removeVideoEffectOutput:error:`

### Method signature changes (6)

- `SCContentFilter -includedApplications`
  - old: `() → NSArray<SCRunningApplication *> *`
  - new: `() → NSArray<SCRunningApplication *> * _Nonnull`
- `SCContentFilter -includedDisplays`
  - old: `() → NSArray<SCDisplay *> *`
  - new: `() → NSArray<SCDisplay *> * _Nonnull`
- `SCContentFilter -includedWindows`
  - old: `() → NSArray<SCWindow *> *`
  - new: `() → NSArray<SCWindow *> * _Nonnull`
- `SCStream -synchronizationClock`
  - old: `() → CMClockRef`
  - new: `() → CMClockRef _Nullable`
- `SCStreamConfiguration -setStreamName:`
  - old: `(NSString *) → void`
  - new: `(NSString * _Nullable) → void`
- `SCStreamConfiguration -streamName`
  - old: `() → NSString *`
  - new: `() → NSString * _Nullable`

### Protocols added (2)

- `SCClipBufferingOutputDelegate`
- `SCRecordingEditorDelegate`

### Enums added (2)

- `SCRecordingEditorMode`
- `task_shared_region_stubs_t`

### Enum members added (8)

- `SCStreamErrorCode.SCStreamErrorInsufficientStorage`
- `SCStreamErrorCode.SCStreamErrorMissingBackgroundMode`
- `SCStreamErrorCode.SCStreamErrorNotSupported`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Externs added (1)

- `SCStreamFrameInfoVideoOrientation`

## ScreenSaver

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ScreenTime

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ScriptingBridge

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## Security

### Enums added (2)

- `_anon_kSecCSMaxSignatures`
- `task_shared_region_stubs_t`

### Enum members added (14)

- `SecCSFlags.kSecCSUseClassicalSignature`
- `SecCSFlags.kSecCSUsePostQuantumSignature`
- `SecCSFlags.kSecCSUseSignature1`
- `SecCSFlags.kSecCSUseSignature2`
- `_anon_errSecCSUnimplemented.errSecCSDetachedCertificates`
- `_anon_errSecCSUnimplemented.errSecCSMultipleSelfSigning`
- `_anon_errSecCSUnimplemented.errSecCSRemoteSignerFirstSlotFull`
- `_anon_errSecCSUnimplemented.errSecCSRemoteSignerSecondSlotFull`
- `_anon_errSecCSUnimplemented.errSecCSUnsupportedAlgorithm`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Function signature changes (108)

- `AuthorizationRightGet`
  - old: `(const char * _Nonnull, CF_RETURNS_RETAINED CFDictionaryRef  _Nullable *) → OSStatus`
  - new: `(const char * _Nonnull, CFDictionaryRef  _Nullable *) → OSStatus`
- `CMSDecoderCopyAllCerts`
  - old: `(CMSDecoderRef _Nonnull, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(CMSDecoderRef _Nonnull, CFArrayRef  _Nullable *) → OSStatus`
- `CMSDecoderCopyContent`
  - old: `(CMSDecoderRef _Nonnull, CF_RETURNS_RETAINED CFDataRef  _Nullable *) → OSStatus`
  - new: `(CMSDecoderRef _Nonnull, CFDataRef  _Nullable *) → OSStatus`
- `CMSDecoderCopyDetachedContent`
  - old: `(CMSDecoderRef _Nonnull, CF_RETURNS_RETAINED CFDataRef  _Nullable *) → OSStatus`
  - new: `(CMSDecoderRef _Nonnull, CFDataRef  _Nullable *) → OSStatus`
- `CMSDecoderCopyEncapsulatedContentType`
  - old: `(CMSDecoderRef _Nonnull, CF_RETURNS_RETAINED CFDataRef  _Nullable *) → OSStatus`
  - new: `(CMSDecoderRef _Nonnull, CFDataRef  _Nullable *) → OSStatus`
- `CMSDecoderCopySignerCert`
  - old: `(CMSDecoderRef _Nonnull, size_t, CF_RETURNS_RETAINED SecCertificateRef  _Nullable *) → OSStatus`
  - new: `(CMSDecoderRef _Nonnull, size_t, SecCertificateRef  _Nullable *) → OSStatus`
- `CMSDecoderCopySignerEmailAddress`
  - old: `(CMSDecoderRef _Nonnull, size_t, CF_RETURNS_RETAINED CFStringRef  _Nullable *) → OSStatus`
  - new: `(CMSDecoderRef _Nonnull, size_t, CFStringRef  _Nullable *) → OSStatus`
- `CMSDecoderCopySignerStatus`
  - old: `(CMSDecoderRef _Nonnull, size_t, CFTypeRef _Nonnull, Boolean, CMSSignerStatus * _Nullable, CF_RETURNS_RETAINED SecTrustRef  _Nullable *, OSStatus * _Nullable) → OSStatus`
  - new: `(CMSDecoderRef _Nonnull, size_t, CFTypeRef _Nonnull, Boolean, CMSSignerStatus * _Nullable, SecTrustRef  _Nullable *, OSStatus * _Nullable) → OSStatus`
- `CMSDecoderCopySignerTimestampCertificates`
  - old: `(CMSDecoderRef _Nonnull, size_t, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(CMSDecoderRef _Nonnull, size_t, CFArrayRef  _Nullable *) → OSStatus`
- `CMSDecoderCreate`
  - old: `(CF_RETURNS_RETAINED CMSDecoderRef  _Nullable *) → OSStatus`
  - new: `(CMSDecoderRef  _Nullable *) → OSStatus`
- `CMSEncode`
  - old: `(CFTypeRef _Nullable, CFTypeRef _Nullable, const SecAsn1Oid * _Nullable, Boolean, CMSSignedAttributes, const void * _Nonnull, size_t, CF_RETURNS_RETAINED CFDataRef  _Nullable *) → OSStatus`
  - new: `(CFTypeRef _Nullable, CFTypeRef _Nullable, const SecAsn1Oid * _Nullable, Boolean, CMSSignedAttributes, const void * _Nonnull, size_t, CFDataRef  _Nullable *) → OSStatus`
- `CMSEncodeContent`
  - old: `(CFTypeRef _Nullable, CFTypeRef _Nullable, CFTypeRef _Nullable, Boolean, CMSSignedAttributes, const void * _Nonnull, size_t, CF_RETURNS_RETAINED CFDataRef  _Nullable *) → OSStatus`
  - new: `(CFTypeRef _Nullable, CFTypeRef _Nullable, CFTypeRef _Nullable, Boolean, CMSSignedAttributes, const void * _Nonnull, size_t, CFDataRef  _Nullable *) → OSStatus`
- `CMSEncoderCopyEncapsulatedContentType`
  - old: `(CMSEncoderRef _Nonnull, CF_RETURNS_RETAINED CFDataRef  _Nullable *) → OSStatus`
  - new: `(CMSEncoderRef _Nonnull, CFDataRef  _Nullable *) → OSStatus`
- `CMSEncoderCopyEncodedContent`
  - old: `(CMSEncoderRef _Nonnull, CF_RETURNS_RETAINED CFDataRef  _Nullable *) → OSStatus`
  - new: `(CMSEncoderRef _Nonnull, CFDataRef  _Nullable *) → OSStatus`
- `CMSEncoderCopyRecipients`
  - old: `(CMSEncoderRef _Nonnull, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(CMSEncoderRef _Nonnull, CFArrayRef  _Nullable *) → OSStatus`
- `CMSEncoderCopySigners`
  - old: `(CMSEncoderRef _Nonnull, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(CMSEncoderRef _Nonnull, CFArrayRef  _Nullable *) → OSStatus`
- `CMSEncoderCopySupportingCerts`
  - old: `(CMSEncoderRef _Nonnull, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(CMSEncoderRef _Nonnull, CFArrayRef  _Nullable *) → OSStatus`
- `CMSEncoderCreate`
  - old: `(CF_RETURNS_RETAINED CMSEncoderRef  _Nullable *) → OSStatus`
  - new: `(CMSEncoderRef  _Nullable *) → OSStatus`
- `SSLCopyCertificateAuthorities`
  - old: `(SSLContextRef _Nonnull, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(SSLContextRef _Nonnull, CFArrayRef  _Nullable *) → OSStatus`
- `SSLCopyDistinguishedNames`
  - old: `(SSLContextRef _Nonnull, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(SSLContextRef _Nonnull, CFArrayRef  _Nullable *) → OSStatus`
- `SSLCopyPeerCertificates`
  - old: `(SSLContextRef _Nonnull, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(SSLContextRef _Nonnull, CFArrayRef  _Nullable *) → OSStatus`
- `SSLCopyPeerTrust`
  - old: `(SSLContextRef _Nonnull, CF_RETURNS_RETAINED SecTrustRef  _Nullable *) → OSStatus`
  - new: `(SSLContextRef _Nonnull, SecTrustRef  _Nullable *) → OSStatus`
- `SSLCopyTrustedRoots`
  - old: `(SSLContextRef _Nonnull, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(SSLContextRef _Nonnull, CFArrayRef  _Nullable *) → OSStatus`
- `SSLNewContext`
  - old: `(Boolean, CF_RETURNS_RETAINED SSLContextRef  _Nullable *) → OSStatus`
  - new: `(Boolean, SSLContextRef  _Nullable *) → OSStatus`
- `SecACLCopyContents`
  - old: `(SecACLRef _Nonnull, CF_RETURNS_RETAINED CFArrayRef  _Nullable *, CF_RETURNS_RETAINED CFStringRef  _Nullable *, SecKeychainPromptSelector * _Nonnull) → OSStatus`
  - new: `(SecACLRef _Nonnull, CFArrayRef  _Nullable *, CFStringRef  _Nullable *, SecKeychainPromptSelector * _Nonnull) → OSStatus`
- `SecACLCopySimpleContents`
  - old: `(SecACLRef _Nonnull, CF_RETURNS_RETAINED CFArrayRef  _Nullable *, CF_RETURNS_RETAINED CFStringRef  _Nullable *, CSSM_ACL_KEYCHAIN_PROMPT_SELECTOR * _Nonnull) → OSStatus`
  - new: `(SecACLRef _Nonnull, CFArrayRef  _Nullable *, CFStringRef  _Nullable *, CSSM_ACL_KEYCHAIN_PROMPT_SELECTOR * _Nonnull) → OSStatus`
- `SecACLCreateFromSimpleContents`
  - old: `(SecAccessRef _Nonnull, CFArrayRef _Nullable, CFStringRef _Nonnull, const CSSM_ACL_KEYCHAIN_PROMPT_SELECTOR * _Nonnull, CF_RETURNS_RETAINED SecACLRef  _Nullable *) → OSStatus`
  - new: `(SecAccessRef _Nonnull, CFArrayRef _Nullable, CFStringRef _Nonnull, const CSSM_ACL_KEYCHAIN_PROMPT_SELECTOR * _Nonnull, SecACLRef  _Nullable *) → OSStatus`
- `SecACLCreateWithSimpleContents`
  - old: `(SecAccessRef _Nonnull, CFArrayRef _Nullable, CFStringRef _Nonnull, SecKeychainPromptSelector, CF_RETURNS_RETAINED SecACLRef  _Nullable *) → OSStatus`
  - new: `(SecAccessRef _Nonnull, CFArrayRef _Nullable, CFStringRef _Nonnull, SecKeychainPromptSelector, SecACLRef  _Nullable *) → OSStatus`
- `SecAccessCopyACLList`
  - old: `(SecAccessRef _Nonnull, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(SecAccessRef _Nonnull, CFArrayRef  _Nullable *) → OSStatus`
- `SecAccessCopyOwnerAndACL`
  - old: `(SecAccessRef _Nonnull, uid_t * _Nullable, gid_t * _Nullable, SecAccessOwnerType * _Nullable, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(SecAccessRef _Nonnull, uid_t * _Nullable, gid_t * _Nullable, SecAccessOwnerType * _Nullable, CFArrayRef  _Nullable *) → OSStatus`
- `SecAccessCopySelectedACLList`
  - old: `(SecAccessRef _Nonnull, CSSM_ACL_AUTHORIZATION_TAG, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(SecAccessRef _Nonnull, CSSM_ACL_AUTHORIZATION_TAG, CFArrayRef  _Nullable *) → OSStatus`
- `SecAccessCreate`
  - old: `(CFStringRef _Nonnull, CFArrayRef _Nullable, CF_RETURNS_RETAINED SecAccessRef  _Nullable *) → OSStatus`
  - new: `(CFStringRef _Nonnull, CFArrayRef _Nullable, SecAccessRef  _Nullable *) → OSStatus`
- `SecAccessCreateFromOwnerAndACL`
  - old: `(const CSSM_ACL_OWNER_PROTOTYPE * _Nonnull, uint32, const CSSM_ACL_ENTRY_INFO * _Nonnull, CF_RETURNS_RETAINED SecAccessRef  _Nullable *) → OSStatus`
  - new: `(const CSSM_ACL_OWNER_PROTOTYPE * _Nonnull, uint32, const CSSM_ACL_ENTRY_INFO * _Nonnull, SecAccessRef  _Nullable *) → OSStatus`
- `SecCertificateCopyCommonName`
  - old: `(SecCertificateRef _Nonnull, CF_RETURNS_RETAINED CFStringRef  _Nullable *) → OSStatus`
  - new: `(SecCertificateRef _Nonnull, CFStringRef  _Nullable *) → OSStatus`
- `SecCertificateCopyEmailAddresses`
  - old: `(SecCertificateRef _Nonnull, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(SecCertificateRef _Nonnull, CFArrayRef  _Nullable *) → OSStatus`
- `SecCertificateCopyKey`
  - old: `(SecCertificateRef _Nonnull) → CF_RETURNS_RETAINED SecKeyRef`
  - new: `(SecCertificateRef _Nonnull) → SecKeyRef`
- `SecCertificateCopyPreference`
  - old: `(CFStringRef _Nonnull, uint32, CF_RETURNS_RETAINED SecCertificateRef  _Nullable *) → OSStatus`
  - new: `(CFStringRef _Nonnull, uint32, SecCertificateRef  _Nullable *) → OSStatus`
- `SecCertificateCopyPublicKey`
  - old: `(SecCertificateRef _Nonnull, CF_RETURNS_RETAINED SecKeyRef  _Nullable *) → OSStatus`
  - new: `(SecCertificateRef _Nonnull, SecKeyRef  _Nullable *) → OSStatus`
- `SecCertificateCreateFromData`
  - old: `(const SecAsn1Item * _Nonnull, CSSM_CERT_TYPE, CSSM_CERT_ENCODING, CF_RETURNS_RETAINED SecCertificateRef  _Nullable *) → OSStatus`
  - new: `(const SecAsn1Item * _Nonnull, CSSM_CERT_TYPE, CSSM_CERT_ENCODING, SecCertificateRef  _Nullable *) → OSStatus`
- `SecCodeCopyDesignatedRequirement`
  - old: `(SecStaticCodeRef _Nonnull, SecCSFlags, CF_RETURNS_RETAINED SecRequirementRef  _Nullable *) → OSStatus`
  - new: `(SecStaticCodeRef _Nonnull, SecCSFlags, SecRequirementRef  _Nullable *) → OSStatus`
- `SecCodeCopyGuestWithAttributes`
  - old: `(SecCodeRef _Nullable, CFDictionaryRef _Nullable, SecCSFlags, CF_RETURNS_RETAINED SecCodeRef  _Nullable *) → OSStatus`
  - new: `(SecCodeRef _Nullable, CFDictionaryRef _Nullable, SecCSFlags, SecCodeRef  _Nullable *) → OSStatus`
- `SecCodeCopyHost`
  - old: `(SecCodeRef _Nonnull, SecCSFlags, CF_RETURNS_RETAINED SecCodeRef  _Nullable *) → OSStatus`
  - new: `(SecCodeRef _Nonnull, SecCSFlags, SecCodeRef  _Nullable *) → OSStatus`
- `SecCodeCopyPath`
  - old: `(SecStaticCodeRef _Nonnull, SecCSFlags, CF_RETURNS_RETAINED CFURLRef  _Nullable *) → OSStatus`
  - new: `(SecStaticCodeRef _Nonnull, SecCSFlags, CFURLRef  _Nullable *) → OSStatus`
- `SecCodeCopySelf`
  - old: `(SecCSFlags, CF_RETURNS_RETAINED SecCodeRef  _Nullable *) → OSStatus`
  - new: `(SecCSFlags, SecCodeRef  _Nullable *) → OSStatus`
- `SecCodeCopySigningInformation`
  - old: `(SecStaticCodeRef _Nonnull, SecCSFlags, CF_RETURNS_RETAINED CFDictionaryRef  _Nullable *) → OSStatus`
  - new: `(SecStaticCodeRef _Nonnull, SecCSFlags, CFDictionaryRef  _Nullable *) → OSStatus`
- `SecCodeCopyStaticCode`
  - old: `(SecCodeRef _Nonnull, SecCSFlags, CF_RETURNS_RETAINED SecStaticCodeRef  _Nullable *) → OSStatus`
  - new: `(SecCodeRef _Nonnull, SecCSFlags, SecStaticCodeRef  _Nullable *) → OSStatus`
- `SecCodeCreateWithXPCMessage`
  - old: `(xpc_object_t _Nonnull, SecCSFlags, CF_RETURNS_RETAINED SecCodeRef  _Nullable *) → OSStatus`
  - new: `(xpc_object_t _Nonnull, SecCSFlags, SecCodeRef  _Nullable *) → OSStatus`
- `SecIdentityCopyCertificate`
  - old: `(SecIdentityRef _Nonnull, CF_RETURNS_RETAINED SecCertificateRef  _Nullable *) → OSStatus`
  - new: `(SecIdentityRef _Nonnull, SecCertificateRef  _Nullable *) → OSStatus`
- `SecIdentityCopyPreference`
  - old: `(CFStringRef _Nonnull, CSSM_KEYUSE, CFArrayRef _Nullable, CF_RETURNS_RETAINED SecIdentityRef  _Nullable *) → OSStatus`
  - new: `(CFStringRef _Nonnull, CSSM_KEYUSE, CFArrayRef _Nullable, SecIdentityRef  _Nullable *) → OSStatus`
- `SecIdentityCopyPrivateKey`
  - old: `(SecIdentityRef _Nonnull, CF_RETURNS_RETAINED SecKeyRef  _Nullable *) → OSStatus`
  - new: `(SecIdentityRef _Nonnull, SecKeyRef  _Nullable *) → OSStatus`
- `SecIdentityCopySystemIdentity`
  - old: `(CFStringRef _Nonnull, CF_RETURNS_RETAINED SecIdentityRef  _Nullable *, CF_RETURNS_RETAINED CFStringRef  _Nullable *) → OSStatus`
  - new: `(CFStringRef _Nonnull, SecIdentityRef  _Nullable *, CFStringRef  _Nullable *) → OSStatus`
- `SecIdentityCreate`
  - old: `(CFAllocatorRef _Nullable, SecCertificateRef _Nonnull, SecKeyRef _Nonnull) → CF_RETURNS_RETAINED SecIdentityRef`
  - new: `(CFAllocatorRef _Nullable, SecCertificateRef _Nonnull, SecKeyRef _Nonnull) → SecIdentityRef`
- `SecIdentityCreateWithCertificate`
  - old: `(CFTypeRef _Nullable, SecCertificateRef _Nonnull, CF_RETURNS_RETAINED SecIdentityRef  _Nullable *) → OSStatus`
  - new: `(CFTypeRef _Nullable, SecCertificateRef _Nonnull, SecIdentityRef  _Nullable *) → OSStatus`
- `SecIdentitySearchCopyNext`
  - old: `(SecIdentitySearchRef _Nonnull, CF_RETURNS_RETAINED SecIdentityRef  _Nullable *) → OSStatus`
  - new: `(SecIdentitySearchRef _Nonnull, SecIdentityRef  _Nullable *) → OSStatus`
- `SecIdentitySearchCreate`
  - old: `(CFTypeRef _Nullable, CSSM_KEYUSE, CF_RETURNS_RETAINED SecIdentitySearchRef  _Nullable *) → OSStatus`
  - new: `(CFTypeRef _Nullable, CSSM_KEYUSE, SecIdentitySearchRef  _Nullable *) → OSStatus`
- `SecItemAdd`
  - old: `(CFDictionaryRef _Nonnull, CF_RETURNS_RETAINED CFTypeRef  _Nullable *) → OSStatus`
  - new: `(CFDictionaryRef _Nonnull, CFTypeRef  _Nullable *) → OSStatus`
- `SecItemCopyMatching`
  - old: `(CFDictionaryRef _Nonnull, CF_RETURNS_RETAINED CFTypeRef  _Nullable *) → OSStatus`
  - new: `(CFDictionaryRef _Nonnull, CFTypeRef  _Nullable *) → OSStatus`
- `SecItemExport`
  - old: `(CFTypeRef _Nonnull, SecExternalFormat, SecItemImportExportFlags, const SecItemImportExportKeyParameters * _Nullable, CF_RETURNS_RETAINED CFDataRef  _Nullable *) → OSStatus`
  - new: `(CFTypeRef _Nonnull, SecExternalFormat, SecItemImportExportFlags, const SecItemImportExportKeyParameters * _Nullable, CFDataRef  _Nullable *) → OSStatus`
- `SecItemImport`
  - old: `(CFDataRef _Nonnull, CFStringRef _Nullable, SecExternalFormat * _Nullable, SecExternalItemType * _Nullable, SecItemImportExportFlags, const SecItemImportExportKeyParameters * _Nullable, SecKeychainRef _Nullable, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(CFDataRef _Nonnull, CFStringRef _Nullable, SecExternalFormat * _Nullable, SecExternalItemType * _Nullable, SecItemImportExportFlags, const SecItemImportExportKeyParameters * _Nullable, SecKeychainRef _Nullable, CFArrayRef  _Nullable *) → OSStatus`
- `SecKeyCreatePair`
  - old: `(SecKeychainRef _Nullable, CSSM_ALGORITHMS, uint32, CSSM_CC_HANDLE, CSSM_KEYUSE, uint32, CSSM_KEYUSE, uint32, SecAccessRef _Nullable, CF_RETURNS_RETAINED SecKeyRef  _Nullable *, CF_RETURNS_RETAINED SecKeyRef  _Nullable *) → OSStatus`
  - new: `(SecKeychainRef _Nullable, CSSM_ALGORITHMS, uint32, CSSM_CC_HANDLE, CSSM_KEYUSE, uint32, CSSM_KEYUSE, uint32, SecAccessRef _Nullable, SecKeyRef  _Nullable *, SecKeyRef  _Nullable *) → OSStatus`
- `SecKeyDeriveFromPassword`
  - old: `(CFStringRef _Nonnull, CFDictionaryRef _Nonnull, CFErrorRef  _Nullable * _Nullable) → CF_RETURNS_RETAINED SecKeyRef`
  - new: `(CFStringRef _Nonnull, CFDictionaryRef _Nonnull, CFErrorRef  _Nullable * _Nullable) → SecKeyRef`
- `SecKeyGenerate`
  - old: `(SecKeychainRef _Nullable, CSSM_ALGORITHMS, uint32, CSSM_CC_HANDLE, CSSM_KEYUSE, uint32, SecAccessRef _Nullable, CF_RETURNS_RETAINED SecKeyRef  _Nullable *) → OSStatus`
  - new: `(SecKeychainRef _Nullable, CSSM_ALGORITHMS, uint32, CSSM_CC_HANDLE, CSSM_KEYUSE, uint32, SecAccessRef _Nullable, SecKeyRef  _Nullable *) → OSStatus`
- `SecKeyGeneratePair`
  - old: `(CFDictionaryRef _Nonnull, CF_RETURNS_RETAINED SecKeyRef  _Nullable *, CF_RETURNS_RETAINED SecKeyRef  _Nullable *) → OSStatus`
  - new: `(CFDictionaryRef _Nonnull, SecKeyRef  _Nullable *, SecKeyRef  _Nullable *) → OSStatus`
- `SecKeyGenerateSymmetric`
  - old: `(CFDictionaryRef _Nonnull, CFErrorRef  _Nullable * _Nullable) → CF_RETURNS_RETAINED SecKeyRef`
  - new: `(CFDictionaryRef _Nonnull, CFErrorRef  _Nullable * _Nullable) → SecKeyRef`
- `SecKeychainAddGenericPassword`
  - old: `(SecKeychainRef _Nullable, UInt32, const char * _Nullable, UInt32, const char * _Nullable, UInt32, const void * _Nonnull, CF_RETURNS_RETAINED SecKeychainItemRef  _Nullable *) → OSStatus`
  - new: `(SecKeychainRef _Nullable, UInt32, const char * _Nullable, UInt32, const char * _Nullable, UInt32, const void * _Nonnull, SecKeychainItemRef  _Nullable *) → OSStatus`
- `SecKeychainAddInternetPassword`
  - old: `(SecKeychainRef _Nullable, UInt32, const char * _Nullable, UInt32, const char * _Nullable, UInt32, const char * _Nullable, UInt32, const char * _Nullable, UInt16, SecProtocolType, SecAuthenticationType, UInt32, const void * _Nonnull, CF_RETURNS_RETAINED SecKeychainItemRef  _Nullable *) → OSStatus`
  - new: `(SecKeychainRef _Nullable, UInt32, const char * _Nullable, UInt32, const char * _Nullable, UInt32, const char * _Nullable, UInt32, const char * _Nullable, UInt16, SecProtocolType, SecAuthenticationType, UInt32, const void * _Nonnull, SecKeychainItemRef  _Nullable *) → OSStatus`
- `SecKeychainCopyAccess`
  - old: `(SecKeychainRef _Nullable, CF_RETURNS_RETAINED SecAccessRef  _Nullable *) → OSStatus`
  - new: `(SecKeychainRef _Nullable, SecAccessRef  _Nullable *) → OSStatus`
- `SecKeychainCopyDefault`
  - old: `(CF_RETURNS_RETAINED SecKeychainRef  _Nullable *) → OSStatus`
  - new: `(SecKeychainRef  _Nullable *) → OSStatus`
- `SecKeychainCopyDomainDefault`
  - old: `(SecPreferencesDomain, CF_RETURNS_RETAINED SecKeychainRef  _Nullable *) → OSStatus`
  - new: `(SecPreferencesDomain, SecKeychainRef  _Nullable *) → OSStatus`
- `SecKeychainCopyDomainSearchList`
  - old: `(SecPreferencesDomain, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(SecPreferencesDomain, CFArrayRef  _Nullable *) → OSStatus`
- `SecKeychainCopySearchList`
  - old: `(CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(CFArrayRef  _Nullable *) → OSStatus`
- `SecKeychainCreate`
  - old: `(const char * _Nonnull, UInt32, const void * _Nullable, Boolean, SecAccessRef _Nullable, CF_RETURNS_RETAINED SecKeychainRef  _Nullable *) → OSStatus`
  - new: `(const char * _Nonnull, UInt32, const void * _Nullable, Boolean, SecAccessRef _Nullable, SecKeychainRef  _Nullable *) → OSStatus`
- `SecKeychainFindGenericPassword`
  - old: `(CFTypeRef _Nullable, UInt32, const char * _Nullable, UInt32, const char * _Nullable, UInt32 * _Nullable, void * _Nullable * _Nullable, CF_RETURNS_RETAINED SecKeychainItemRef  _Nullable *) → OSStatus`
  - new: `(CFTypeRef _Nullable, UInt32, const char * _Nullable, UInt32, const char * _Nullable, UInt32 * _Nullable, void * _Nullable * _Nullable, SecKeychainItemRef  _Nullable *) → OSStatus`
- `SecKeychainFindInternetPassword`
  - old: `(CFTypeRef _Nullable, UInt32, const char * _Nullable, UInt32, const char * _Nullable, UInt32, const char * _Nullable, UInt32, const char * _Nullable, UInt16, SecProtocolType, SecAuthenticationType, UInt32 * _Nullable, void * _Nullable * _Nullable, CF_RETURNS_RETAINED SecKeychainItemRef  _Nullable *) → OSStatus`
  - new: `(CFTypeRef _Nullable, UInt32, const char * _Nullable, UInt32, const char * _Nullable, UInt32, const char * _Nullable, UInt32, const char * _Nullable, UInt16, SecProtocolType, SecAuthenticationType, UInt32 * _Nullable, void * _Nullable * _Nullable, SecKeychainItemRef  _Nullable *) → OSStatus`
- `SecKeychainItemCopyAccess`
  - old: `(SecKeychainItemRef _Nonnull, CF_RETURNS_RETAINED SecAccessRef  _Nullable *) → OSStatus`
  - new: `(SecKeychainItemRef _Nonnull, SecAccessRef  _Nullable *) → OSStatus`
- `SecKeychainItemCopyFromPersistentReference`
  - old: `(CFDataRef _Nonnull, CF_RETURNS_RETAINED SecKeychainItemRef  _Nullable *) → OSStatus`
  - new: `(CFDataRef _Nonnull, SecKeychainItemRef  _Nullable *) → OSStatus`
- `SecKeychainItemCopyKeychain`
  - old: `(SecKeychainItemRef _Nonnull, CF_RETURNS_RETAINED SecKeychainRef  _Nullable *) → OSStatus`
  - new: `(SecKeychainItemRef _Nonnull, SecKeychainRef  _Nullable *) → OSStatus`
- `SecKeychainItemCreateCopy`
  - old: `(SecKeychainItemRef _Nonnull, SecKeychainRef _Nullable, SecAccessRef _Nullable, CF_RETURNS_RETAINED SecKeychainItemRef  _Nullable *) → OSStatus`
  - new: `(SecKeychainItemRef _Nonnull, SecKeychainRef _Nullable, SecAccessRef _Nullable, SecKeychainItemRef  _Nullable *) → OSStatus`
- `SecKeychainItemCreateFromContent`
  - old: `(SecItemClass, SecKeychainAttributeList * _Nonnull, UInt32, const void * _Nullable, SecKeychainRef _Nullable, SecAccessRef _Nullable, CF_RETURNS_RETAINED SecKeychainItemRef  _Nullable *) → OSStatus`
  - new: `(SecItemClass, SecKeychainAttributeList * _Nonnull, UInt32, const void * _Nullable, SecKeychainRef _Nullable, SecAccessRef _Nullable, SecKeychainItemRef  _Nullable *) → OSStatus`
- `SecKeychainItemCreatePersistentReference`
  - old: `(SecKeychainItemRef _Nonnull, CF_RETURNS_RETAINED CFDataRef  _Nullable *) → OSStatus`
  - new: `(SecKeychainItemRef _Nonnull, CFDataRef  _Nullable *) → OSStatus`
- `SecKeychainItemExport`
  - old: `(CFTypeRef _Nonnull, SecExternalFormat, SecItemImportExportFlags, const SecKeyImportExportParameters * _Nullable, CF_RETURNS_RETAINED CFDataRef  _Nullable *) → OSStatus`
  - new: `(CFTypeRef _Nonnull, SecExternalFormat, SecItemImportExportFlags, const SecKeyImportExportParameters * _Nullable, CFDataRef  _Nullable *) → OSStatus`
- `SecKeychainItemImport`
  - old: `(CFDataRef _Nonnull, CFStringRef _Nullable, SecExternalFormat * _Nullable, SecExternalItemType * _Nullable, SecItemImportExportFlags, const SecKeyImportExportParameters * _Nullable, SecKeychainRef _Nullable, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(CFDataRef _Nonnull, CFStringRef _Nullable, SecExternalFormat * _Nullable, SecExternalItemType * _Nullable, SecItemImportExportFlags, const SecKeyImportExportParameters * _Nullable, SecKeychainRef _Nullable, CFArrayRef  _Nullable *) → OSStatus`
- `SecKeychainOpen`
  - old: `(const char * _Nonnull, CF_RETURNS_RETAINED SecKeychainRef  _Nullable *) → OSStatus`
  - new: `(const char * _Nonnull, SecKeychainRef  _Nullable *) → OSStatus`
- `SecKeychainSearchCopyNext`
  - old: `(SecKeychainSearchRef _Nonnull, CF_RETURNS_RETAINED SecKeychainItemRef  _Nullable *) → OSStatus`
  - new: `(SecKeychainSearchRef _Nonnull, SecKeychainItemRef  _Nullable *) → OSStatus`
- `SecKeychainSearchCreateFromAttributes`
  - old: `(CFTypeRef _Nullable, SecItemClass, const SecKeychainAttributeList * _Nullable, CF_RETURNS_RETAINED SecKeychainSearchRef  _Nullable *) → OSStatus`
  - new: `(CFTypeRef _Nullable, SecItemClass, const SecKeychainAttributeList * _Nullable, SecKeychainSearchRef  _Nullable *) → OSStatus`
- `SecPKCS12Import`
  - old: `(CFDataRef _Nonnull, CFDictionaryRef _Nonnull, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(CFDataRef _Nonnull, CFDictionaryRef _Nonnull, CFArrayRef  _Nullable *) → OSStatus`
- `SecPolicySearchCopyNext`
  - old: `(SecPolicySearchRef _Nonnull, CF_RETURNS_RETAINED SecPolicyRef  _Nullable *) → OSStatus`
  - new: `(SecPolicySearchRef _Nonnull, SecPolicyRef  _Nullable *) → OSStatus`
- `SecPolicySearchCreate`
  - old: `(CSSM_CERT_TYPE, const SecAsn1Oid * _Nonnull, const SecAsn1Item * _Nullable, CF_RETURNS_RETAINED SecPolicySearchRef  _Nullable *) → OSStatus`
  - new: `(CSSM_CERT_TYPE, const SecAsn1Oid * _Nonnull, const SecAsn1Item * _Nullable, SecPolicySearchRef  _Nullable *) → OSStatus`
- `SecRequirementCopyData`
  - old: `(SecRequirementRef _Nonnull, SecCSFlags, CF_RETURNS_RETAINED CFDataRef  _Nullable *) → OSStatus`
  - new: `(SecRequirementRef _Nonnull, SecCSFlags, CFDataRef  _Nullable *) → OSStatus`
- `SecRequirementCopyString`
  - old: `(SecRequirementRef _Nonnull, SecCSFlags, CF_RETURNS_RETAINED CFStringRef  _Nullable *) → OSStatus`
  - new: `(SecRequirementRef _Nonnull, SecCSFlags, CFStringRef  _Nullable *) → OSStatus`
- `SecRequirementCreateWithData`
  - old: `(CFDataRef _Nonnull, SecCSFlags, CF_RETURNS_RETAINED SecRequirementRef  _Nullable *) → OSStatus`
  - new: `(CFDataRef _Nonnull, SecCSFlags, SecRequirementRef  _Nullable *) → OSStatus`
- `SecRequirementCreateWithString`
  - old: `(CFStringRef _Nonnull, SecCSFlags, CF_RETURNS_RETAINED SecRequirementRef  _Nullable *) → OSStatus`
  - new: `(CFStringRef _Nonnull, SecCSFlags, SecRequirementRef  _Nullable *) → OSStatus`
- `SecRequirementCreateWithStringAndErrors`
  - old: `(CFStringRef _Nonnull, SecCSFlags, CFErrorRef  _Nullable * _Nullable, CF_RETURNS_RETAINED SecRequirementRef  _Nullable *) → OSStatus`
  - new: `(CFStringRef _Nonnull, SecCSFlags, CFErrorRef  _Nullable * _Nullable, SecRequirementRef  _Nullable *) → OSStatus`
- `SecStaticCodeCreateWithPath`
  - old: `(CFURLRef _Nonnull, SecCSFlags, CF_RETURNS_RETAINED SecStaticCodeRef  _Nullable *) → OSStatus`
  - new: `(CFURLRef _Nonnull, SecCSFlags, SecStaticCodeRef  _Nullable *) → OSStatus`
- `SecStaticCodeCreateWithPathAndAttributes`
  - old: `(CFURLRef _Nonnull, SecCSFlags, CFDictionaryRef _Nonnull, CF_RETURNS_RETAINED SecStaticCodeRef  _Nullable *) → OSStatus`
  - new: `(CFURLRef _Nonnull, SecCSFlags, CFDictionaryRef _Nonnull, SecStaticCodeRef  _Nullable *) → OSStatus`
- `SecTrustCopyAnchorCertificates`
  - old: `(CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(CFArrayRef  _Nullable *) → OSStatus`
- `SecTrustCopyCertificateChain`
  - old: `(SecTrustRef _Nonnull) → CF_RETURNS_RETAINED CFArrayRef`
  - new: `(SecTrustRef _Nonnull) → CFArrayRef`
- `SecTrustCopyCustomAnchorCertificates`
  - old: `(SecTrustRef _Nonnull, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(SecTrustRef _Nonnull, CFArrayRef  _Nullable *) → OSStatus`
- `SecTrustCopyPolicies`
  - old: `(SecTrustRef _Nonnull, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(SecTrustRef _Nonnull, CFArrayRef  _Nullable *) → OSStatus`
- `SecTrustCreateWithCertificates`
  - old: `(CFTypeRef _Nonnull, CFTypeRef _Nullable, CF_RETURNS_RETAINED SecTrustRef  _Nullable *) → OSStatus`
  - new: `(CFTypeRef _Nonnull, CFTypeRef _Nullable, SecTrustRef  _Nullable *) → OSStatus`
- `SecTrustEvaluateWithError`
  - old: `(SecTrustRef _Nonnull, CF_RETURNS_RETAINED CFErrorRef  _Nullable *) → bool`
  - new: `(SecTrustRef _Nonnull, CFErrorRef  _Nullable *) → bool`
- `SecTrustGetResult`
  - old: `(SecTrustRef _Nonnull, SecTrustResultType * _Nullable, CF_RETURNS_RETAINED CFArrayRef  _Nullable *, CSSM_TP_APPLE_EVIDENCE_INFO * _Nullable * _Nullable) → OSStatus`
  - new: `(SecTrustRef _Nonnull, SecTrustResultType * _Nullable, CFArrayRef  _Nullable *, CSSM_TP_APPLE_EVIDENCE_INFO * _Nullable * _Nullable) → OSStatus`
- `SecTrustSettingsCopyCertificates`
  - old: `(SecTrustSettingsDomain, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(SecTrustSettingsDomain, CFArrayRef  _Nullable *) → OSStatus`
- `SecTrustSettingsCopyModificationDate`
  - old: `(SecCertificateRef _Nonnull, SecTrustSettingsDomain, CF_RETURNS_RETAINED CFDateRef  _Nullable *) → OSStatus`
  - new: `(SecCertificateRef _Nonnull, SecTrustSettingsDomain, CFDateRef  _Nullable *) → OSStatus`
- `SecTrustSettingsCopyTrustSettings`
  - old: `(SecCertificateRef _Nonnull, SecTrustSettingsDomain, CF_RETURNS_RETAINED CFArrayRef  _Nullable *) → OSStatus`
  - new: `(SecCertificateRef _Nonnull, SecTrustSettingsDomain, CFArrayRef  _Nullable *) → OSStatus`
- `SecTrustSettingsCreateExternalRepresentation`
  - old: `(SecTrustSettingsDomain, CF_RETURNS_RETAINED CFDataRef  _Nullable *) → OSStatus`
  - new: `(SecTrustSettingsDomain, CFDataRef  _Nullable *) → OSStatus`
- `SecTrustedApplicationCopyData`
  - old: `(SecTrustedApplicationRef _Nonnull, CF_RETURNS_RETAINED CFDataRef  _Nullable *) → OSStatus`
  - new: `(SecTrustedApplicationRef _Nonnull, CFDataRef  _Nullable *) → OSStatus`
- `SecTrustedApplicationCreateFromPath`
  - old: `(const char * _Nullable, CF_RETURNS_RETAINED SecTrustedApplicationRef  _Nullable *) → OSStatus`
  - new: `(const char * _Nullable, SecTrustedApplicationRef  _Nullable *) → OSStatus`

### Externs added (6)

- `CSSMOID_HYBRID_COMPOSITE_MLDSA87_RSA3072_WithSHA512`
- `CSSMOID_HYBRID_COMPOSITE_MLDSA87_RSA3072_WithSHA512_draft_13`
- `kSecCFErrorDetachedCertificates`
- `kSecCodeInfoChosenSignature`
- `kSecCodeInfoSignerInfoSKID`
- `kSecCodeInfoTotalSignatures`

## SecurityFoundation

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## SecurityHI

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## SecurityInterface

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## SecurityUI

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## SensitiveContentAnalysis

### Methods added (1)

- `SCSensitivityAnalysis -detectedTypes`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Externs added (2)

- `SCSensitiveContentTypeGoreOrViolence`
- `SCSensitiveContentTypeSexuallyExplicit`

## SensorKit

### Classes added (3)

- `SRHeadphoneSettings`
- `SRHeadphoneSettingsHearingAssistance`
- `SRSourceDevice`

### Methods added (1)

- `SRFetchResult -sourceDevice`

### Enums added (4)

- `SRHeadphoneSettingsAdaptiveAudioStrength`
- `SRHeadphoneSettingsListeningMode`
- `SRHeadphoneSettingsSettingEnablement`
- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Externs added (2)

- `SRSensorHeadphoneMotion`
- `SRSensorHeadphoneSettings`

## ServiceManagement

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## SharedFileList

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## SharedWithYou

### Methods added (2)

- `SWCollaborationView -pendingAccessRequestsCount`
- `SWCollaborationView -setPendingAccessRequestsCount:`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Externs added (1)

- `SWCopyRepresentationTypeIdentifier`

## SharedWithYouCore

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ShazamKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## Social

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## SoundAnalysis

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## Speech

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## SpeechRecognition

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## SpeechSynthesis

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Function signature changes (2)

- `CopyPhonemesFromText`
  - old: `(SpeechChannel _Nonnull, CFStringRef _Nonnull, CF_RETURNS_RETAINED CFStringRef  _Nullable *) → OSErr`
  - new: `(SpeechChannel _Nonnull, CFStringRef _Nonnull, CFStringRef  _Nullable *) → OSErr`
- `CopySpeechProperty`
  - old: `(SpeechChannel _Nonnull, CFStringRef _Nonnull, CF_RETURNS_RETAINED CFTypeRef  _Nullable *) → OSErr`
  - new: `(SpeechChannel _Nonnull, CFStringRef _Nonnull, CFTypeRef  _Nullable *) → OSErr`

## SpriteKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## StickerKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## StoreKit

### Method signature changes (3)

- `SKDownload -contentLength`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT NSNumber *`
  - new: `() → NSNumber *`
- `SKProduct -contentLengths`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT NSArray<NSNumber *> *`
  - new: `() → NSArray<NSNumber *> *`
- `SKProduct -contentVersion`
  - old: `() → API_DEPRECATED_WITH_REPLACEMENT NSString *`
  - new: `() → NSString *`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## SwiftUI

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## SwiftUICore

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## Symbols

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## SyncServices

### Classes added (1)

- `ISyncRecordReference`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Deprecation changes (2)

- `ISyncSession: deprecated (none) → 10.7`
- `ISyncSessionDriver: deprecated 10.7 → (none)`

## SystemConfiguration

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## SystemExtensions

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## ThreadNetwork

### Methods added (2)

- `THClient -enableCredentialSharingModeForExtendedPANID:completion:`
- `THClient -retrieveActiveCredentialsForNearbyNetworksWithCompletion:`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## Translation

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## UniformTypeIdentifiers

### Methods added (1)

- `UTType +typeWithIdentifier:allowUndeclared:`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Externs added (1)

- `UTTypeMarkdown`

## UserNotifications

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## UserNotificationsUI

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## VideoSubscriberAccount

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## VideoToolbox

### Methods added (5)

- `VTLowLatencyFrameInterpolationConfiguration +maximumDimensionForSpatialScaleFactor:`
- `VTLowLatencyFrameInterpolationConfiguration +maximumPixelCountForSpatialScaleFactor:`
- `VTLowLatencySuperResolutionScalerConfiguration +maximumDimensionForSpatialScaleFactor:`
- `VTLowLatencySuperResolutionScalerConfiguration +maximumPixelCountForSpatialScaleFactor:`
- `VTLowLatencySuperResolutionScalerConfiguration +supportedScaleFactors`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (6)

- `_anon_kVTPropertyNotSupportedErr.kVTLogTransferFunctionMismatchErr`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Externs added (4)

- `kVTCompressionPreset_ConsistentQuality`
- `kVTCompressionPropertyKey_ConstantQualityFactor`
- `kVTCompressionPropertyKey_LogTransferFunction`
- `kVTProjectionKind_AppleImmersiveVideo`

## Virtualization

### Classes added (21)

- `VZCustomVirtioDevice`
- `VZCustomVirtioDeviceConfiguration`
- `VZCustomVirtioDeviceDelegateProvider`
- `VZCustomVirtioDeviceProvider`
- `VZEFISignature`
- `VZEFISignatureDatabaseConfiguration`
- `VZEFISignatureList`
- `VZEFISignatureSHA256Hash`
- `VZEFISignatureX509Certificate`
- `VZGuestMemoryMapping`
- `VZGuestProvisioningOptions`
- `VZMacGuestProvisioningOptions`
- `VZNegotiatedVirtioFeatureSet`
- `VZUSBPassthroughDevice`
- `VZUSBPassthroughDeviceConfiguration`
- `VZVirtioDeviceSpecificConfiguration`
- `VZVirtioFeatureSet`
- `VZVirtioQueue`
- `VZVirtioQueueElement`
- `VZVirtioSharedMemoryRegion`
- `VZVirtioSharedMemoryRegionConfiguration`

### Methods added (16)

- `VZEFIVariableStore -disableSecureBootWithError:`
- `VZEFIVariableStore -enableSecureBootUsingDefaultPlatformKeyWithError:`
- `VZEFIVariableStore -enableSecureBootWithPlatformKey:error:`
- `VZEFIVariableStore -enrollDefaultSecureBootSignaturesWithError:`
- `VZEFIVariableStore -enrollSecureBootSignatures:error:`
- `VZEFIVariableStore -getEnrolledSecureBootSignaturesWithError:`
- `VZEFIVariableStore -getSecureBootEnabled:error:`
- `VZEFIVariableStore -resetSecureBootWithError:`
- `VZMacOSVirtualMachineStartOptions -guestProvisioningOptions`
- `VZMacOSVirtualMachineStartOptions -setGuestProvisioningOptions:error:`
- `VZUSBController -delegate`
- `VZUSBController -setDelegate:`
- `VZVirtualMachineConfiguration -customVirtioDevices`
- `VZVirtualMachineConfiguration -label`
- `VZVirtualMachineConfiguration -setCustomVirtioDevices:`
- `VZVirtualMachineConfiguration -setLabel:`

### Protocols added (3)

- `VZCustomVirtioDeviceConfigurationDelegate`
- `VZCustomVirtioDeviceDelegate`
- `VZUSBControllerDelegate`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (10)

- `VZErrorCode.VZErrorEFISecureBootEnrollmentFailed`
- `VZErrorCode.VZErrorEFIVariableInaccessible`
- `VZErrorCode.VZErrorGuestProvisioningInvalidFullName`
- `VZErrorCode.VZErrorGuestProvisioningInvalidPassword`
- `VZErrorCode.VZErrorGuestProvisioningInvalidUsername`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## Vision

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (7)

- `VNErrorCode.VNErrorResourceCorrupted`
- `VNErrorCode.VNErrorResourceUnavailable`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## WebKit

### Classes added (5)

- `WKContentWorldConfiguration`
- `WKDOMNodeSnapshot`
- `WKFormInfo`
- `WKImmersiveEnvironment`
- `WKJSHandle`

### Methods added (17)

- `WKContentWorld +worldWithConfiguration:`
- `WKHTTPCookieStore -getCookiesForURL:completionHandler:`
- `WKNavigationAction -mainFrameNavigation`
- `WKNavigationResponse -mainFrameNavigation`
- `WKUserContentController -addBuffer:name:contentWorld:`
- `WKUserContentController -removeBufferWithName:contentWorld:`
- `WKWebView -loadURL:`
- `WKWebView -refreshController`
- `WKWebView -setRefreshController:`
- `WKWebpagePreferences -allowsJSHandleCreationInPageWorld`
- `WKWebpagePreferences -alternateRequest`
- `WKWebpagePreferences -globalPrivacyControlEnabled`
- `WKWebpagePreferences -overrideReferrer`
- `WKWebpagePreferences -setAllowsJSHandleCreationInPageWorld:`
- `WKWebpagePreferences -setAlternateRequest:`
- `WKWebpagePreferences -setGlobalPrivacyControlEnabled:`
- `WKWebpagePreferences -setOverrideReferrer:`

### Protocols added (1)

- `WKImmersiveEnvironmentDelegate`

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

### Enum base type changes (1)

- `nw_link_quality_t`
  - old: `int32`
  - new: `uint8`

## WidgetKit

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## iTunesLibrary

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## vImage

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## vecLib

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`

## vmnet

### Enums added (1)

- `task_shared_region_stubs_t`

### Enum members added (5)

- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_COPY_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_EXTRACT_DENIED`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_FIRST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_LAST`
- `virtual_memory_guard_exception_code_t.kGUARD_EXC_COW_DEFEATURED_SHARE_MAP_AS_COPY_DENIED`
