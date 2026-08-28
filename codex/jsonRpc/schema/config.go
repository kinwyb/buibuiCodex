// Code generated from JSON Schema using quicktype. DO NOT EDIT.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    schema, err := UnmarshalSchema(bytes)
//    bytes, err = schema.Marshal()

package schema

import "bytes"
import "errors"

import "encoding/json"

func UnmarshalSchema(data []byte) (Schema, error) {
	var r Schema
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *Schema) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// Base config deserialized from ~/.codex/config.toml.
type Schema struct {
	// Agent-related settings (thread limits, etc.).
	Agents map[string]*AgentValue `json:"agents,omitempty"`
	// Whether the model may request a login shell for shell-based tools. Default to `true`
	//
	// If `true`, the model may request a login shell (`login = true`), and omitting `login`
	// defaults to using a login shell. If `false`, the model can never use a login shell:
	// `login = true` requests are rejected, and omitting `login` defaults to a non-login shell.
	AllowLoginShell *bool `json:"allow_login_shell,omitempty"`
	// When `false`, disables analytics across Codex product surfaces in this machine. Defaults
	// to `true`.
	Analytics *AnalyticsConfigToml `json:"analytics,omitempty"`
	// Default approval policy for executing commands.
	ApprovalPolicy *AskForApproval `json:"approval_policy"`
	// Configures who approval requests are routed to for review once they have been escalated.
	// This does not disable separate safety checks such as ARC.
	ApprovalsReviewer *ApprovalsReviewer `json:"approvals_reviewer,omitempty"`
	// Settings for app-specific controls.
	Apps map[string]AppConfig `json:"apps,omitempty"`
	// Optional product SKU forwarded on host-owned Codex Apps MCP requests.
	AppsMCPProductSku *string `json:"apps_mcp_product_sku,omitempty"`
	// Machine-local realtime audio device preferences used by realtime voice.
	Audio *RealtimeAudioToml `json:"audio,omitempty"`
	// Optional policy instructions for the guardian auto-reviewer.
	AutoReview *AutoReviewToml `json:"auto_review,omitempty"`
	// Maximum poll window for background terminal output (`write_stdin`), in milliseconds.
	// Default: `300000` (5 minutes).
	BackgroundTerminalMaxTimeout *int64                `json:"background_terminal_max_timeout,omitempty"`
	BrowserUse                   *BrowserUseConfigToml `json:"browser_use,omitempty"`
	// Base URL for requests to ChatGPT (as opposed to the OpenAI API).
	ChatgptBaseURL *string `json:"chatgpt_base_url,omitempty"`
	// When `true`, checks for Codex updates on startup and surfaces update prompts. Set to
	// `false` only if your Codex updates are centrally managed. Defaults to `true`.
	CheckForUpdateOnStartup *bool `json:"check_for_update_on_startup,omitempty"`
	// Preferred backend for storing CLI auth credentials. file (default): Use a file in the
	// Codex home directory. keyring: Use an OS-specific keyring service. auto: Use the keyring
	// if available, otherwise use a file.
	CLIAuthCredentialsStore *AuthCredentialsStoreMode `json:"cli_auth_credentials_store,omitempty"`
	// Compact prompt used for history compaction.
	CompactPrompt *string                `json:"compact_prompt,omitempty"`
	ComputerUse   *ComputerUseConfigToml `json:"computer_use,omitempty"`
	// Default permissions profile to apply. Names starting with `:` refer to built-in profiles;
	// other names are resolved from the `[permissions]` table.
	DefaultPermissions *string `json:"default_permissions,omitempty"`
	// Opaque desktop settings stored alongside the rest of config.toml.
	Desktop map[string]interface{} `json:"desktop,omitempty"`
	// Developer instructions inserted as a `developer` role message.
	DeveloperInstructions *string `json:"developer_instructions,omitempty"`
	// When true, disables burst-paste detection for typed input entirely. All characters are
	// inserted as they are received, and no buffering or placeholder replacement will occur for
	// fast keypress bursts.
	DisablePasteBurst             *bool   `json:"disable_paste_burst,omitempty"`
	ExperimentalCompactPromptFile *string `json:"experimental_compact_prompt_file,omitempty"`
	// Experimental / do not use. Replaces the built-in realtime start instructions inserted
	// into developer messages when realtime becomes active.
	ExperimentalRealtimeStartInstructions *string `json:"experimental_realtime_start_instructions,omitempty"`
	// Experimental / do not use. Overrides only the WebRTC realtime call creation base URL.
	// This is separate from `experimental_realtime_ws_base_url` because WebRTC call creation is
	// HTTP, while sideband control is websocket.
	ExperimentalRealtimeWebrtcCallBaseURL *string `json:"experimental_realtime_webrtc_call_base_url,omitempty"`
	// Experimental / do not use. Overrides only the realtime conversation websocket transport
	// instructions (the `Op::RealtimeConversation` `/ws` session.update instructions) without
	// changing normal prompts.
	ExperimentalRealtimeWsBackendPrompt *string `json:"experimental_realtime_ws_backend_prompt,omitempty"`
	// Experimental / do not use. Overrides only the realtime conversation websocket transport
	// base URL (the `Op::RealtimeConversation` `/v1/realtime` connection) without changing
	// normal provider HTTP requests.
	ExperimentalRealtimeWsBaseURL *string `json:"experimental_realtime_ws_base_url,omitempty"`
	// Experimental / do not use. Selects the realtime websocket model/snapshot used for the
	// `Op::RealtimeConversation` connection.
	ExperimentalRealtimeWsModel *string `json:"experimental_realtime_ws_model,omitempty"`
	// Experimental / do not use. Replaces the synthesized realtime startup context appended to
	// websocket session instructions. An empty string disables startup context injection
	// entirely.
	ExperimentalRealtimeWsStartupContext *string `json:"experimental_realtime_ws_startup_context,omitempty"`
	// Experimental / do not use. Selects the thread store implementation.
	ExperimentalThreadStore        *ThreadStoreToml `json:"experimental_thread_store,omitempty"`
	ExperimentalUseUnifiedExecTool *bool            `json:"experimental_use_unified_exec_tool,omitempty"`
	// Centralized feature flags (new). Prefer this over individual toggles.
	Features *SchemaFeatures `json:"features,omitempty"`
	// When `false`, disables feedback collection across Codex product surfaces. Defaults to
	// `true`.
	Feedback *FeedbackConfigToml `json:"feedback,omitempty"`
	// Optional URI-based file opener. If set, citations to files in the model output will be
	// hyperlinked using the specified URI scheme.
	FileOpener *URIBasedFileOpener `json:"file_opener,omitempty"`
	// When set, restricts ChatGPT login to one or more workspace identifiers.
	ForcedChatgptWorkspaceID *ForcedChatgptWorkspaceIDS `json:"forced_chatgpt_workspace_id"`
	// When set, restricts the login mechanism users may use.
	ForcedLoginMethod *ForcedLoginMethod `json:"forced_login_method,omitempty"`
	// Compatibility-only settings retained so legacy `ghost_snapshot` config still loads.
	GhostSnapshot *GhostSnapshotToml `json:"ghost_snapshot,omitempty"`
	// Goal-related settings.
	Goals *GoalsToml `json:"goals,omitempty"`
	// When set to `true`, `AgentReasoning` events will be hidden from the UI/output. Defaults
	// to `false`.
	HideAgentReasoning *bool `json:"hide_agent_reasoning,omitempty"`
	// Settings that govern if and what will be written to `~/.codex/history.jsonl`.
	History *History `json:"history,omitempty"`
	// Lifecycle hooks configured inline in TOML plus user-level overrides.
	Hooks *HooksToml `json:"hooks,omitempty"`
	// Whether to inject the `<apps_instructions>` developer block.
	IncludeAppsInstructions *bool `json:"include_apps_instructions,omitempty"`
	// Whether to inject the `<collaboration_mode>` developer block.
	IncludeCollaborationModeInstructions *bool `json:"include_collaboration_mode_instructions,omitempty"`
	// Whether to inject the `<environment_context>` user block.
	IncludeEnvironmentContext *bool `json:"include_environment_context,omitempty"`
	// Whether to inject the `<permissions instructions>` developer block.
	IncludePermissionsInstructions *bool `json:"include_permissions_instructions,omitempty"`
	// System instructions.
	Instructions *string `json:"instructions,omitempty"`
	// Directory where Codex writes log files. Setting this value explicitly also enables the
	// TUI text log in this directory. Defaults to `$CODEX_HOME/log`.
	LogDir *string `json:"log_dir,omitempty"`
	// User-level marketplace entries keyed by marketplace name.
	Marketplaces map[string]MarketplaceConfig `json:"marketplaces,omitempty"`
	// Optional fixed port for the local HTTP callback server used during MCP OAuth login. When
	// unset, Codex will bind to an ephemeral port chosen by the OS.
	MCPOauthCallbackPort *int64 `json:"mcp_oauth_callback_port,omitempty"`
	// Optional redirect URI to use during MCP OAuth login. When set, this URI is used in the
	// OAuth authorization request instead of the local listener address. The local callback
	// listener still binds to 127.0.0.1 (using `mcp_oauth_callback_port` when provided).
	MCPOauthCallbackURL *string `json:"mcp_oauth_callback_url,omitempty"`
	// Preferred backend for storing MCP OAuth credentials. keyring: Use an OS-specific keyring
	// service. https://github.com/openai/codex/blob/main/codex-rs/rmcp-client/src/oauth.rs#L2
	// file: Use a file in the Codex home directory. auto (default): Use the OS-specific keyring
	// service if available, otherwise use a file.
	MCPOauthCredentialsStore *OAuthCredentialsStoreMode `json:"mcp_oauth_credentials_store,omitempty"`
	// Definition for MCP servers that Codex can reach out to for tool calls.
	MCPServers map[string]RawMCPServerConfig `json:"mcp_servers,omitempty"`
	// Memories subsystem settings.
	Memories *MemoriesToml `json:"memories,omitempty"`
	// Optional override of model selection.
	Model *string `json:"model,omitempty"`
	// Token usage threshold triggering auto-compaction of conversation history.
	ModelAutoCompactTokenLimit *int64 `json:"model_auto_compact_token_limit,omitempty"`
	// Controls whether the auto-compaction limit applies to the full context or only to tokens
	// after the carried prefix in the current compaction window.
	ModelAutoCompactTokenLimitScope *AutoCompactTokenLimitScope `json:"model_auto_compact_token_limit_scope,omitempty"`
	// Optional path to a JSON model catalog (applied on startup only). Per-thread `config`
	// overrides are accepted but do not reapply this (no-ops).
	ModelCatalogJSON *string `json:"model_catalog_json,omitempty"`
	// Size of the context window for the model, in tokens.
	ModelContextWindow *int64 `json:"model_context_window,omitempty"`
	// Optional path to a file containing model instructions that will override the built-in
	// instructions for the selected model. Users are STRONGLY DISCOURAGED from using this
	// field, as deviating from the instructions sanctioned by Codex will likely degrade model
	// performance.
	ModelInstructionsFile *string `json:"model_instructions_file,omitempty"`
	// Provider to use from the model_providers map.
	ModelProvider *string `json:"model_provider,omitempty"`
	// User-defined provider entries that extend the built-in list. Built-in IDs cannot be
	// overridden.
	ModelProviders        map[string]ModelProviderInfo `json:"model_providers,omitempty"`
	ModelReasoningEffort  *string                      `json:"model_reasoning_effort,omitempty"`
	ModelReasoningSummary *ReasoningSummary            `json:"model_reasoning_summary,omitempty"`
	// Optional verbosity control for GPT-5 models (Responses API `text.verbosity`).
	ModelVerbosity *Verbosity `json:"model_verbosity,omitempty"`
	// Collection of in-product notices (different from notifications) See
	// [`crate::types::Notice`] for more details
	Notice *Notice `json:"notice,omitempty"`
	// Optional external command to spawn for end-user notifications.
	Notify []string `json:"notify,omitempty"`
	// Base URL override for the built-in `openai` model provider.
	OpenaiBaseURL *string `json:"openai_base_url,omitempty"`
	// Orchestrator-owned feature settings.
	Orchestrator *OrchestratorToml `json:"orchestrator,omitempty"`
	// Preferred OSS provider for local models, e.g. "lmstudio" or "ollama".
	OSSProvider *string `json:"oss_provider,omitempty"`
	// OTEL configuration.
	Otel *OtelConfigToml `json:"otel,omitempty"`
	// Named permissions profiles.
	Permissions map[string]interface{} `json:"permissions,omitempty"`
	// Optionally specify a personality for the model
	Personality             *Personality `json:"personality,omitempty"`
	PlanModeReasoningEffort *string      `json:"plan_mode_reasoning_effort,omitempty"`
	// User-level plugin config entries keyed by plugin name.
	Plugins map[string]PluginConfig `json:"plugins,omitempty"`
	// Profile to use from the `profiles` map.
	Profile *string `json:"profile,omitempty"`
	// Named profiles to facilitate switching between different configurations.
	Profiles map[string]ConfigProfile `json:"profiles,omitempty"`
	// Ordered list of fallback filenames to look for when AGENTS.md is missing.
	ProjectDocFallbackFilenames []string `json:"project_doc_fallback_filenames,omitempty"`
	// Maximum total bytes of project instruction content across all selected environments.
	ProjectDocMaxBytes *int64 `json:"project_doc_max_bytes,omitempty"`
	// Markers used to detect the project root when searching parent directories for `.codex`
	// folders. Defaults to [".git"] when unset.
	ProjectRootMarkers []string                 `json:"project_root_markers,omitempty"`
	Projects           map[string]ProjectConfig `json:"projects,omitempty"`
	// Experimental / do not use. Realtime websocket session selection. `version` controls v1/v2
	// and `type` controls conversational/transcription.
	Realtime *RealtimeToml `json:"realtime,omitempty"`
	// Bounded, product-owned metadata attached to every Responses API request.
	ResponsesAPIMetadata map[string]string `json:"responses_api_metadata,omitempty"`
	// Review model override used by the `/review` feature.
	ReviewModel *string `json:"review_model,omitempty"`
	// Sandbox mode to use.
	SandboxMode *SandboxMode `json:"sandbox_mode,omitempty"`
	// Sandbox configuration to apply if `sandbox` is `WorkspaceWrite`.
	SandboxWorkspaceWrite *SandboxWorkspaceWrite `json:"sandbox_workspace_write,omitempty"`
	// Optional explicit service tier request id for new turns (for example `default`,
	// `priority`, or `flex`; legacy `fast` also works).
	ServiceTier            *string                     `json:"service_tier,omitempty"`
	ShellEnvironmentPolicy *ShellEnvironmentPolicyToml `json:"shell_environment_policy,omitempty"`
	// When set to `true`, `AgentReasoningRawContentEvent` events will be shown in the
	// UI/output. Defaults to `false`.
	ShowRawAgentReasoning *bool `json:"show_raw_agent_reasoning,omitempty"`
	// User-level skill config entries keyed by SKILL.md path.
	Skills *SkillsConfig `json:"skills,omitempty"`
	// Directory where Codex stores the SQLite state DB. Defaults to `$CODEX_SQLITE_HOME` when
	// set. Otherwise uses `$CODEX_HOME`.
	SqliteHome *string `json:"sqlite_home,omitempty"`
	// Suppress warnings about unstable (under development) features.
	SuppressUnstableFeaturesWarning *bool `json:"suppress_unstable_features_warning,omitempty"`
	// Token budget applied when storing tool/function outputs in the context manager.
	ToolOutputTokenLimit *int64 `json:"tool_output_token_limit,omitempty"`
	// Additional discoverable tools that can be suggested for installation.
	ToolSuggest *ToolSuggestConfig `json:"tool_suggest,omitempty"`
	// Nested tools section for feature toggles
	Tools *ToolsToml `json:"tools,omitempty"`
	// Collection of settings that are specific to the TUI.
	Tui *Tui `json:"tui,omitempty"`
	// Controls the web search tool mode: disabled, cached, indexed, or live.
	WebSearch *WebSearchMode `json:"web_search,omitempty"`
	// Windows-specific configuration.
	Windows *WindowsToml `json:"windows,omitempty"`
}

type AgentRoleToml struct {
	// Path to a role-specific config layer. Relative paths are resolved relative to the
	// `config.toml` that defines them.
	ConfigFile *string `json:"config_file,omitempty"`
	// Human-facing role documentation used in spawn tool guidance. Required unless supplied by
	// the referenced agent role file.
	Description *string `json:"description,omitempty"`
	// Candidate nicknames for agents spawned with this role.
	NicknameCandidates []string `json:"nickname_candidates,omitempty"`
}

// When `false`, disables analytics across Codex product surfaces in this machine. Defaults
// to `true`.
//
// Analytics settings loaded from config.toml. Fields are optional so we can apply defaults.
type AnalyticsConfigToml struct {
	// When `false`, disables analytics across Codex product surfaces in this profile.
	Enabled *bool `json:"enabled,omitempty"`
}

// Fine-grained controls for individual approval flows.
//
// When a field is `true`, commands in that category are allowed. When it is `false`, those
// requests are automatically rejected instead of shown to the user.
type AskForApprovalClass struct {
	Granular GranularApprovalConfig `json:"granular"`
}

type GranularApprovalConfig struct {
	// Whether to allow MCP elicitation prompts.
	MCPElicitations bool `json:"mcp_elicitations"`
	// Whether to allow prompts triggered by the `request_permissions` tool.
	RequestPermissions *bool `json:"request_permissions,omitempty"`
	// Whether to allow prompts triggered by execpolicy `prompt` rules.
	Rules bool `json:"rules"`
	// Whether to allow shell command approval requests, including inline
	// `with_additional_permissions` and `require_escalated` requests.
	SandboxApproval bool `json:"sandbox_approval"`
	// Whether to allow approval prompts triggered by skill script execution.
	SkillApproval *bool `json:"skill_approval,omitempty"`
}

// Default settings for all apps.
//
// Default settings that apply to all apps.
//
// Config values for a single app/connector.
type AppConfig struct {
	// Reviewer for approval prompts unless overridden by per-app settings.
	//
	// Reviewer for approval prompts from this app, overriding the thread default.
	ApprovalsReviewer *ApprovalsReviewer `json:"approvals_reviewer,omitempty"`
	// Approval mode for tools unless overridden by per-app or per-tool settings.
	//
	// Approval mode for tools in this app unless a tool override exists.
	DefaultToolsApprovalMode *AppToolApproval `json:"default_tools_approval_mode,omitempty"`
	// Whether tools with `destructive_hint = true` are allowed by default.
	//
	// Whether tools with `destructive_hint = true` are allowed for this app.
	DestructiveEnabled *bool `json:"destructive_enabled,omitempty"`
	// When `false`, apps are disabled unless overridden by per-app settings.
	//
	// When `false`, Codex does not surface this app.
	Enabled *bool `json:"enabled,omitempty"`
	// Whether tools with `open_world_hint = true` are allowed by default.
	//
	// Whether tools with `open_world_hint = true` are allowed for this app.
	OpenWorldEnabled *bool `json:"open_world_enabled,omitempty"`
	// Whether tools are enabled by default for this app.
	DefaultToolsEnabled *bool `json:"default_tools_enabled,omitempty"`
	// Per-tool settings for this app.
	Tools map[string]AppToolConfig `json:"tools,omitempty"`
}

// Per-tool settings for a single app tool.
type AppToolConfig struct {
	// Approval mode for this tool.
	ApprovalMode *AppToolApproval `json:"approval_mode,omitempty"`
	// Whether this tool is enabled. `Some(true)` explicitly allows this tool.
	Enabled *bool `json:"enabled,omitempty"`
}

// Machine-local realtime audio device preferences used by realtime voice.
type RealtimeAudioToml struct {
	Microphone *string `json:"microphone,omitempty"`
	Speaker    *string `json:"speaker,omitempty"`
}

// Optional policy instructions for the guardian auto-reviewer.
type AutoReviewToml struct {
	// Additional policy instructions inserted into the guardian prompt.
	Policy *string `json:"policy,omitempty"`
}

type BrowserUseConfigToml struct {
	AllowHistoryAccess  *bool                                       `json:"allow_history_access,omitempty"`
	DefaultOriginPolicy *BrowserUseOriginPolicyConfigToml           `json:"default_origin_policy,omitempty"`
	Origins             map[string]BrowserUseOriginPolicyConfigToml `json:"origins,omitempty"`
}

type BrowserUseOriginPolicyConfigToml struct {
	Access        *Toml `json:"access,omitempty"`
	Downloads     *Toml `json:"downloads,omitempty"`
	FullCDPAccess *Toml `json:"full_cdp_access,omitempty"`
	Uploads       *Toml `json:"uploads,omitempty"`
}

type ComputerUseConfigToml struct {
	DefaultAppAccess *Toml                         `json:"default_app_access,omitempty"`
	Macos            *ComputerUseMacosConfigToml   `json:"macos,omitempty"`
	Windows          *ComputerUseWindowsConfigToml `json:"windows,omitempty"`
}

type ComputerUseMacosConfigToml struct {
	BundleIDS map[string]Toml `json:"bundle_ids,omitempty"`
}

type ComputerUseWindowsConfigToml struct {
	Aumids map[string]Toml                   `json:"aumids,omitempty"`
	Exes   []ComputerUseWindowsExeConfigToml `json:"exes,omitempty"`
}

type ComputerUseWindowsExeConfigToml struct {
	Access        Toml    `json:"access"`
	BinaryName    *string `json:"binary_name,omitempty"`
	ProductName   string  `json:"product_name"`
	PublisherName string  `json:"publisher_name"`
}

// Experimental / do not use. Selects the thread store implementation.
type ThreadStoreToml struct {
	Type ExperimentalThreadStoreType `json:"type"`
}

// Centralized feature flags (new). Prefer this over individual toggles.
type SchemaFeatures struct {
	ApplyPatchFreeform                  *bool                                            `json:"apply_patch_freeform,omitempty"`
	ApplyPatchPreserveLineEndings       *bool                                            `json:"apply_patch_preserve_line_endings,omitempty"`
	ApplyPatchStreamingEvents           *bool                                            `json:"apply_patch_streaming_events,omitempty"`
	Apps                                *bool                                            `json:"apps,omitempty"`
	AppsMCPPathOverride                 *TentacledAppsMCPPathOverride                    `json:"apps_mcp_path_override"`
	AuthElicitation                     *bool                                            `json:"auth_elicitation,omitempty"`
	BackgroundPaginatedRolloutMigration *bool                                            `json:"background_paginated_rollout_migration,omitempty"`
	BrowserUse                          *bool                                            `json:"browser_use,omitempty"`
	BrowserUseExternal                  *bool                                            `json:"browser_use_external,omitempty"`
	BrowserUseFullCDPAccess             *bool                                            `json:"browser_use_full_cdp_access,omitempty"`
	Chronicle                           *bool                                            `json:"chronicle,omitempty"`
	CodeMode                            *FeatureTomlForCodeModeConfigToml                `json:"code_mode"`
	CodeModeBufferedExec                *bool                                            `json:"code_mode_buffered_exec,omitempty"`
	CodeModeHost                        *FeatureTomlForCodeModeHostConfigToml            `json:"code_mode_host"`
	CodeModeInterrupt                   *bool                                            `json:"code_mode_interrupt,omitempty"`
	CodeModeOnly                        *bool                                            `json:"code_mode_only,omitempty"`
	CodexGitCommit                      *bool                                            `json:"codex_git_commit,omitempty"`
	CodexHooks                          *bool                                            `json:"codex_hooks,omitempty"`
	Collab                              *bool                                            `json:"collab,omitempty"`
	CollaborationModes                  *bool                                            `json:"collaboration_modes,omitempty"`
	CompactionImageBudget               *bool                                            `json:"compaction_image_budget,omitempty"`
	ComputerUse                         *bool                                            `json:"computer_use,omitempty"`
	ConcurrentReasoningSummaries        *bool                                            `json:"concurrent_reasoning_summaries,omitempty"`
	Connectors                          *bool                                            `json:"connectors,omitempty"`
	ContentItemKinds                    *bool                                            `json:"content_item_kinds,omitempty"`
	CurrentTimeReminder                 *FeatureTomlForCurrentTimeReminderConfigToml     `json:"current_time_reminder"`
	CwdRelativeTurnDiffs                *bool                                            `json:"cwd_relative_turn_diffs,omitempty"`
	DefaultModeRequestUserInput         *bool                                            `json:"default_mode_request_user_input,omitempty"`
	DeferredExecutor                    *bool                                            `json:"deferred_executor,omitempty"`
	DeferredToolWorldState              *bool                                            `json:"deferred_tool_world_state,omitempty"`
	ElevatedWindowsSandbox              *bool                                            `json:"elevated_windows_sandbox,omitempty"`
	EnableExperimentalWindowsSandbox    *bool                                            `json:"enable_experimental_windows_sandbox,omitempty"`
	EnableFanout                        *bool                                            `json:"enable_fanout,omitempty"`
	EnableMCPApps                       *bool                                            `json:"enable_mcp_apps,omitempty"`
	EnableRequestCompression            *bool                                            `json:"enable_request_compression,omitempty"`
	ExecPermissionApprovals             *bool                                            `json:"exec_permission_approvals,omitempty"`
	ExecutedToolCallMetadata            *bool                                            `json:"executed_tool_call_metadata,omitempty"`
	ExecutorCapabilityDiscovery         *bool                                            `json:"executor_capability_discovery,omitempty"`
	ExperimentalUseUnifiedExecTool      *bool                                            `json:"experimental_use_unified_exec_tool,omitempty"`
	ExperimentalWindowsSandbox          *bool                                            `json:"experimental_windows_sandbox,omitempty"`
	ExternalAgentMemoryImport           *bool                                            `json:"external_agent_memory_import,omitempty"`
	ExternalMigration                   *bool                                            `json:"external_migration,omitempty"`
	FastMode                            *bool                                            `json:"fast_mode,omitempty"`
	Goals                               *bool                                            `json:"goals,omitempty"`
	GuardianApproval                    *bool                                            `json:"guardian_approval,omitempty"`
	GuardianEnhancedNodeReplTranscripts *bool                                            `json:"guardian_enhanced_node_repl_transcripts,omitempty"`
	GuardianEXT                         *bool                                            `json:"guardian_ext,omitempty"`
	GuardianNodeReplTranscriptImages    *bool                                            `json:"guardian_node_repl_transcript_images,omitempty"`
	GuardianReuseParentCompaction       *bool                                            `json:"guardian_reuse_parent_compaction,omitempty"`
	Guardianv2                          *FeatureTomlForGuardianV2ConfigToml              `json:"guardianv2"`
	Hooks                               *bool                                            `json:"hooks,omitempty"`
	ImageDetailOriginal                 *bool                                            `json:"image_detail_original,omitempty"`
	ImageGeneration                     *bool                                            `json:"image_generation,omitempty"`
	ImageResizeNotice                   *bool                                            `json:"image_resize_notice,omitempty"`
	Imagegenext                         *bool                                            `json:"imagegenext,omitempty"`
	InAppBrowser                        *bool                                            `json:"in_app_browser,omitempty"`
	InAppChat                           *bool                                            `json:"in_app_chat,omitempty"`
	InAppDictation                      *bool                                            `json:"in_app_dictation,omitempty"`
	InAppLocalAutomation                *bool                                            `json:"in_app_local_automation,omitempty"`
	InAppUpdates                        *bool                                            `json:"in_app_updates,omitempty"`
	ItemIDS                             *bool                                            `json:"item_ids,omitempty"`
	JSRepl                              *bool                                            `json:"js_repl,omitempty"`
	JSReplToolsOnly                     *bool                                            `json:"js_repl_tools_only,omitempty"`
	LocalThreadStoreCompression         *bool                                            `json:"local_thread_store_compression,omitempty"`
	MCP2026_07_28                       *bool                                            `json:"mcp_2026_07_28,omitempty"`
	Memories                            *bool                                            `json:"memories,omitempty"`
	MemoryTool                          *bool                                            `json:"memory_tool,omitempty"`
	MentionsV2                          *bool                                            `json:"mentions_v2,omitempty"`
	MultiAgent                          *bool                                            `json:"multi_agent,omitempty"`
	MultiAgentMode                      *bool                                            `json:"multi_agent_mode,omitempty"`
	MultiAgentV2                        *FeatureTomlForMultiAgentV2ConfigToml            `json:"multi_agent_v2"`
	NetworkProxy                        *FeatureTomlForNetworkProxyConfigToml            `json:"network_proxy"`
	NonPrefixedMCPToolNames             *FeatureTomlForNonPrefixedMCPToolNamesConfigToml `json:"non_prefixed_mcp_tool_names"`
	Personality                         *bool                                            `json:"personality,omitempty"`
	PluginHooks                         *bool                                            `json:"plugin_hooks,omitempty"`
	PluginSharing                       *bool                                            `json:"plugin_sharing,omitempty"`
	Plugins                             *bool                                            `json:"plugins,omitempty"`
	PreventIdleSleep                    *bool                                            `json:"prevent_idle_sleep,omitempty"`
	Psp                                 *bool                                            `json:"psp,omitempty"`
	RealtimeConversation                *bool                                            `json:"realtime_conversation,omitempty"`
	RecommendedPlugins                  *bool                                            `json:"recommended_plugins,omitempty"`
	RemoteCompactionV2                  *bool                                            `json:"remote_compaction_v2,omitempty"`
	RemoteControl                       *bool                                            `json:"remote_control,omitempty"`
	RemoteModels                        *bool                                            `json:"remote_models,omitempty"`
	RemotePlugin                        *bool                                            `json:"remote_plugin,omitempty"`
	RequestPermissions                  *bool                                            `json:"request_permissions,omitempty"`
	RequestPermissionsTool              *bool                                            `json:"request_permissions_tool,omitempty"`
	RequestRule                         *bool                                            `json:"request_rule,omitempty"`
	ResizeAllImages                     *bool                                            `json:"resize_all_images,omitempty"`
	RespectSystemProxy                  *bool                                            `json:"respect_system_proxy,omitempty"`
	ResponsesWebsockets                 *bool                                            `json:"responses_websockets,omitempty"`
	ResponsesWebsocketsV2               *bool                                            `json:"responses_websockets_v2,omitempty"`
	RetainClientDeveloperMessages       *bool                                            `json:"retain_client_developer_messages,omitempty"`
	RolloutBudget                       *FeatureTomlForRolloutBudgetConfigToml           `json:"rollout_budget"`
	RuntimeMetrics                      *bool                                            `json:"runtime_metrics,omitempty"`
	SearchTool                          *bool                                            `json:"search_tool,omitempty"`
	SecretAuthStorage                   *bool                                            `json:"secret_auth_storage,omitempty"`
	SendAsyncMessage                    *bool                                            `json:"send_async_message,omitempty"`
	ShellSnapshot                       *bool                                            `json:"shell_snapshot,omitempty"`
	ShellSnapshotV2                     *bool                                            `json:"shell_snapshot_v2,omitempty"`
	ShellTool                           *bool                                            `json:"shell_tool,omitempty"`
	ShellZshFork                        *bool                                            `json:"shell_zsh_fork,omitempty"`
	SkillEnvVarDependencyPrompt         *bool                                            `json:"skill_env_var_dependency_prompt,omitempty"`
	SkillMCPDependencyInstall           *bool                                            `json:"skill_mcp_dependency_install,omitempty"`
	SkillSearch                         *bool                                            `json:"skill_search,omitempty"`
	Sqlite                              *bool                                            `json:"sqlite,omitempty"`
	StandaloneWebSearch                 *bool                                            `json:"standalone_web_search,omitempty"`
	Steer                               *bool                                            `json:"steer,omitempty"`
	Telepathy                           *bool                                            `json:"telepathy,omitempty"`
	TerminalResizeReflow                *bool                                            `json:"terminal_resize_reflow,omitempty"`
	TerminalVisualizationInstructions   *bool                                            `json:"terminal_visualization_instructions,omitempty"`
	TokenBudget                         *FeatureTomlForTokenBudgetConfigToml             `json:"token_budget"`
	ToolCallMCPElicitation              *bool                                            `json:"tool_call_mcp_elicitation,omitempty"`
	ToolRegistry                        *ToolRegistryConfigToml                          `json:"tool_registry,omitempty"`
	ToolSearch                          *bool                                            `json:"tool_search,omitempty"`
	ToolSearchAlwaysDeferMCPTools       *bool                                            `json:"tool_search_always_defer_mcp_tools,omitempty"`
	ToolSuggest                         *bool                                            `json:"tool_suggest,omitempty"`
	TuiAppServer                        *bool                                            `json:"tui_app_server,omitempty"`
	UnavailableDummyTools               *bool                                            `json:"unavailable_dummy_tools,omitempty"`
	UnboundedConnectionRetries          *bool                                            `json:"unbounded_connection_retries,omitempty"`
	Undo                                *bool                                            `json:"undo,omitempty"`
	UnifiedExec                         *bool                                            `json:"unified_exec,omitempty"`
	UnifiedExecZshFork                  *bool                                            `json:"unified_exec_zsh_fork,omitempty"`
	UnifiedImageBudget                  *bool                                            `json:"unified_image_budget,omitempty"`
	UseAgentIdentity                    *bool                                            `json:"use_agent_identity,omitempty"`
	UseLegacyLandlock                   *bool                                            `json:"use_legacy_landlock,omitempty"`
	UseLinuxSandboxBwrap                *bool                                            `json:"use_linux_sandbox_bwrap,omitempty"`
	ViewImage                           *bool                                            `json:"view_image,omitempty"`
	WebSearch                           *bool                                            `json:"web_search,omitempty"`
	WebSearchCached                     *bool                                            `json:"web_search_cached,omitempty"`
	WebSearchRequest                    *bool                                            `json:"web_search_request,omitempty"`
	WorkspaceDependencies               *bool                                            `json:"workspace_dependencies,omitempty"`
	WorkspaceOwnerUsageNudge            *bool                                            `json:"workspace_owner_usage_nudge,omitempty"`
}

type PurpleAppsMCPPathOverride struct {
	Enabled *bool   `json:"enabled,omitempty"`
	Path    *string `json:"path,omitempty"`
}

type CodeModeConfigToml struct {
	// Default yield timeout for code-mode exec calls, in milliseconds.
	DefaultExecYieldTimeMS *int64 `json:"default_exec_yield_time_ms,omitempty"`
	// Exact tool namespaces to expose only as direct model tools. These tools bypass deferral,
	// remain top-level in code-mode-only sessions, and are omitted from the nested code-mode
	// tool surface.
	DirectOnlyToolNamespaces []string `json:"direct_only_tool_namespaces,omitempty"`
	Enabled                  *bool    `json:"enabled,omitempty"`
	// Exact tool namespaces to omit from the code-mode nested tool surface.
	ExcludedToolNamespaces []string `json:"excluded_tool_namespaces,omitempty"`
}

type CodeModeHostConfigToml struct {
	// Keep code mode fail-closed when the standalone host is unavailable.
	DisableInProcessFallback *bool `json:"disable_in_process_fallback,omitempty"`
	Enabled                  *bool `json:"enabled,omitempty"`
}

type CurrentTimeReminderConfigToml struct {
	ClockSource             *CurrentTimeSource               `json:"clock_source,omitempty"`
	DeliveryMode            *CurrentTimeReminderDeliveryMode `json:"delivery_mode,omitempty"`
	Enabled                 *bool                            `json:"enabled,omitempty"`
	ReminderIntervalSeconds *int64                           `json:"reminder_interval_seconds,omitempty"`
	// Expose the input-interruptible `clock.sleep` tool.
	SleepTool *bool `json:"sleep_tool,omitempty"`
}

// User-configurable prompt, approval, and context settings for Guardian v2.
type GuardianV2ConfigToml struct {
	ClassifierInstructions         *string                          `json:"classifier_instructions,omitempty"`
	Enabled                        *bool                            `json:"enabled,omitempty"`
	MaxActionTokens                *int64                           `json:"max_action_tokens,omitempty"`
	MaxClassifierInstructionTokens *int64                           `json:"max_classifier_instruction_tokens,omitempty"`
	MaxParentCompactionTokens      *int64                           `json:"max_parent_compaction_tokens,omitempty"`
	MaxToolCallLag                 *int64                           `json:"max_tool_call_lag,omitempty"`
	ReasoningEffort                *string                          `json:"reasoning_effort,omitempty"`
	ReuseParentCompaction          *bool                            `json:"reuse_parent_compaction,omitempty"`
	ReviewScope                    *GuardianV2ReviewScopeConfigToml `json:"review_scope,omitempty"`
	ReviewThreshold                *float64                         `json:"review_threshold,omitempty"`
	Transcript                     *GuardianV2TranscriptConfigToml  `json:"transcript,omitempty"`
}

// Optional tool-call categories available to the Guardian v2 classifier.
type GuardianV2ReviewScopeConfigToml struct {
	// Restrict asynchronous classification and fast approvals to browser and computer-use tools.
	ComputerUseOnly *bool `json:"computer_use_only,omitempty"`
	// Include sandboxed shell command calls in Guardian v2 classification.
	SandboxedExecCommands *bool `json:"sandboxed_exec_commands,omitempty"`
}

// Bounds and optional sources for the Guardian v2 conversation transcript.
type GuardianV2TranscriptConfigToml struct {
	// Include recent screenshots from messages and configured tool outputs.
	IncludeImages              *bool                        `json:"include_images,omitempty"`
	MaxMessageEntryTokens      *int64                       `json:"max_message_entry_tokens,omitempty"`
	MaxMessageTranscriptTokens *int64                       `json:"max_message_transcript_tokens,omitempty"`
	MaxRecentNonUserEntries    *int64                       `json:"max_recent_non_user_entries,omitempty"`
	MaxToolEntryTokens         *int64                       `json:"max_tool_entry_tokens,omitempty"`
	MaxToolTranscriptTokens    *int64                       `json:"max_tool_transcript_tokens,omitempty"`
	Sources                    []GuardianV2TranscriptSource `json:"sources,omitempty"`
}

type MultiAgentV2ConfigToml struct {
	DefaultWaitTimeoutMS *int64 `json:"default_wait_timeout_ms,omitempty"`
	Enabled              *bool  `json:"enabled,omitempty"`
	// Exposes `model` and `reasoning_effort` on the multi-agent v2 spawn tool and adds
	// corresponding guidance to root and subagent usage hints.
	ExposeSpawnAgentModelOverrides *bool   `json:"expose_spawn_agent_model_overrides,omitempty"`
	HideSpawnAgentMetadata         *bool   `json:"hide_spawn_agent_metadata,omitempty"`
	MaxConcurrentThreadsPerSession *int64  `json:"max_concurrent_threads_per_session,omitempty"`
	MaxWaitTimeoutMS               *int64  `json:"max_wait_timeout_ms,omitempty"`
	MinWaitTimeoutMS               *int64  `json:"min_wait_timeout_ms,omitempty"`
	MultiAgentModeHintText         *string `json:"multi_agent_mode_hint_text,omitempty"`
	NonCodeModeOnly                *bool   `json:"non_code_mode_only,omitempty"`
	RootAgentUsageHintText         *string `json:"root_agent_usage_hint_text,omitempty"`
	// Overrides inherited developer instructions for subagents without role-specific
	// instructions.
	SubagentDeveloperInstructions *string `json:"subagent_developer_instructions,omitempty"`
	SubagentUsageHintText         *string `json:"subagent_usage_hint_text,omitempty"`
	ToolNamespace                 *string `json:"tool_namespace,omitempty"`
	// Deprecated compatibility field. Its value is ignored.
	UsageHintEnabled *bool   `json:"usage_hint_enabled,omitempty"`
	UsageHintText    *string `json:"usage_hint_text,omitempty"`
	// Expose the multi-agent v2 `wait_agent` tool.
	WaitAgentEnabled *bool `json:"wait_agent_enabled,omitempty"`
}

type NetworkProxyConfigToml struct {
	AllowLocalBinding                *bool                 `json:"allow_local_binding,omitempty"`
	AllowUpstreamProxy               *bool                 `json:"allow_upstream_proxy,omitempty"`
	CredentialBroker                 *bool                 `json:"credential_broker,omitempty"`
	DangerouslyAllowAllUnixSockets   *bool                 `json:"dangerously_allow_all_unix_sockets,omitempty"`
	DangerouslyAllowNonLoopbackProxy *bool                 `json:"dangerously_allow_non_loopback_proxy,omitempty"`
	Domains                          map[string]Toml       `json:"domains,omitempty"`
	EnableSocks5                     *bool                 `json:"enable_socks5,omitempty"`
	EnableSocks5UDP                  *bool                 `json:"enable_socks5_udp,omitempty"`
	Enabled                          *bool                 `json:"enabled,omitempty"`
	Mode                             *NetworkProxyModeToml `json:"mode,omitempty"`
	ProxyURL                         *string               `json:"proxy_url,omitempty"`
	SocksURL                         *string               `json:"socks_url,omitempty"`
	UnixSockets                      map[string]Toml       `json:"unix_sockets,omitempty"`
}

type NonPrefixedMCPToolNamesConfigToml struct {
	Enabled *bool `json:"enabled,omitempty"`
	// MCP servers whose tools should omit the legacy `mcp__` namespace prefix.
	ServerNames []string `json:"server_names,omitempty"`
}

type RolloutBudgetConfigToml struct {
	Enabled            *bool    `json:"enabled,omitempty"`
	LimitTokens        *int64   `json:"limit_tokens,omitempty"`
	PrefillTokenWeight *float64 `json:"prefill_token_weight,omitempty"`
	// Remaining weighted-token values that trigger reminders when crossed.
	ReminderAtRemainingTokens []int64  `json:"reminder_at_remaining_tokens,omitempty"`
	SamplingTokenWeight       *float64 `json:"sampling_token_weight,omitempty"`
}

type TokenBudgetConfigToml struct {
	// Additional tokens available after the compaction threshold for fallback note-taking.
	AutoCompactFallbackBufferTokens *int64 `json:"auto_compact_fallback_buffer_tokens,omitempty"`
	// Developer message sampled before an automatic context-window rollover.
	AutoCompactFallbackPrompt *string `json:"auto_compact_fallback_prompt,omitempty"`
	Enabled                   *bool   `json:"enabled,omitempty"`
	// Guidance appended to the context-window metadata in a developer message.
	GuidanceMessage *string `json:"guidance_message,omitempty"`
	// Reminder template. `{n_remaining}` is replaced with the tokens remaining before
	// auto-compaction.
	ReminderMessageTemplate *string `json:"reminder_message_template,omitempty"`
	// Number of tokens remaining before auto-compaction when the wrap-up reminder is emitted.
	ReminderThresholdTokens *int64 `json:"reminder_threshold_tokens,omitempty"`
	// Whether to expose the built-in history and notes extension.
	UseHistoryNotesExtension *bool `json:"use_history_notes_extension,omitempty"`
}

type ToolRegistryConfigToml struct {
	// Fail the turn when multiple tools share the same effective name.
	ErrorOnToolCollisions *bool `json:"error_on_tool_collisions,omitempty"`
	// Include authoritative tool information in per-turn request metadata.
	TurnMetadataIncludesToolInfo *bool `json:"turn_metadata_includes_tool_info,omitempty"`
}

// When `false`, disables feedback collection across Codex product surfaces. Defaults to
// `true`.
type FeedbackConfigToml struct {
	// When `false`, disables the feedback flow across Codex product surfaces.
	Enabled *bool `json:"enabled,omitempty"`
}

// Compatibility-only settings retained so legacy `ghost_snapshot` config still loads.
type GhostSnapshotToml struct {
	// Legacy no-op setting retained for compatibility.
	DisableWarnings *bool `json:"disable_warnings,omitempty"`
	// Legacy no-op setting retained for compatibility.
	IgnoreLargeUntrackedDirs *int64 `json:"ignore_large_untracked_dirs,omitempty"`
	// Legacy no-op setting retained for compatibility.
	IgnoreLargeUntrackedFiles *int64 `json:"ignore_large_untracked_files,omitempty"`
}

// Goal-related settings.
type GoalsToml struct {
	// Maximum token budget allowed for a goal and default budget for new goals.
	MaxGoalTokenBudget *int64 `json:"max_goal_token_budget,omitempty"`
}

// Settings that govern if and what will be written to `~/.codex/history.jsonl`.
type History struct {
	// If set, the maximum size of the history file in bytes. The oldest entries are dropped
	// once the file exceeds this limit.
	MaxBytes *int64 `json:"max_bytes,omitempty"`
	// If true, history entries will not be written to disk.
	Persistence *HistoryPersistence `json:"persistence,omitempty"`
}

// Lifecycle hooks configured inline in TOML plus user-level overrides.
type HooksToml struct {
	PermissionRequest []MatcherGroup           `json:"PermissionRequest,omitempty"`
	PostCompact       []MatcherGroup           `json:"PostCompact,omitempty"`
	PostToolUse       []MatcherGroup           `json:"PostToolUse,omitempty"`
	PreCompact        []MatcherGroup           `json:"PreCompact,omitempty"`
	PreToolUse        []MatcherGroup           `json:"PreToolUse,omitempty"`
	SessionEnd        []MatcherGroup           `json:"SessionEnd,omitempty"`
	SessionStart      []MatcherGroup           `json:"SessionStart,omitempty"`
	State             map[string]HookStateToml `json:"state,omitempty"`
	Stop              []MatcherGroup           `json:"Stop,omitempty"`
	SubagentStart     []MatcherGroup           `json:"SubagentStart,omitempty"`
	SubagentStop      []MatcherGroup           `json:"SubagentStop,omitempty"`
	UserPromptSubmit  []MatcherGroup           `json:"UserPromptSubmit,omitempty"`
}

type MatcherGroup struct {
	Hooks   []HookHandlerConfig `json:"hooks,omitempty"`
	Matcher *string             `json:"matcher,omitempty"`
}

type HookHandlerConfig struct {
	// Approximate token threshold for spilling this hook's `additionalContext` to disk. Unset
	// uses 2,500 tokens; `0` disables spilling for this hook. The threshold is evaluated
	// against the original context; a spilled preview also includes recovery metadata.
	AdditionalContextLimit *int64                 `json:"additionalContextLimit,omitempty"`
	Async                  *bool                  `json:"async,omitempty"`
	Command                *string                `json:"command,omitempty"`
	CommandWindows         *string                `json:"commandWindows,omitempty"`
	StatusMessage          *string                `json:"statusMessage,omitempty"`
	Timeout                *int64                 `json:"timeout,omitempty"`
	Type                   HookHandlerConfigType  `json:"type"`
	Input                  map[string]interface{} `json:"input,omitempty"`
	Server                 *string                `json:"server,omitempty"`
	Tool                   *string                `json:"tool,omitempty"`
}

type HookStateToml struct {
	Enabled     *bool   `json:"enabled,omitempty"`
	TrustedHash *string `json:"trusted_hash,omitempty"`
}

// Raw MCP config shape used for deserialization and supported-field JSON Schema
// generation.
//
// Fields that are accepted only to produce targeted validation errors should be skipped in
// the generated schema.
//
// Keep `TryFrom<RawMcpServerConfig> for McpServerConfig` exhaustively destructuring this
// struct so new TOML fields cannot be added here without updating the validation/mapping
// logic that produces [`McpServerConfig`].
type RawMCPServerConfig struct {
	Args                     []string          `json:"args,omitempty"`
	Auth                     *MCPServerAuth    `json:"auth,omitempty"`
	BearerTokenEnvVar        *string           `json:"bearer_token_env_var,omitempty"`
	Command                  *string           `json:"command,omitempty"`
	Cwd                      *string           `json:"cwd,omitempty"`
	DefaultToolsApprovalMode *AppToolApproval  `json:"default_tools_approval_mode,omitempty"`
	DisabledTools            []string          `json:"disabled_tools,omitempty"`
	Enabled                  *bool             `json:"enabled,omitempty"`
	EnabledTools             []string          `json:"enabled_tools,omitempty"`
	Env                      map[string]string `json:"env,omitempty"`
	EnvHTTPHeaders           map[string]string `json:"env_http_headers,omitempty"`
	EnvVars                  []MCPServerEnvVar `json:"env_vars,omitempty"`
	EnvironmentID            *string           `json:"environment_id,omitempty"`
	HTTPHeaders              map[string]string `json:"http_headers,omitempty"`
	HTTPHeadersHelper        *string           `json:"http_headers_helper,omitempty"`
	// Legacy display-name field accepted for backward compatibility.
	Name                      *string                        `json:"name,omitempty"`
	Oauth                     *MCPServerOAuthConfig          `json:"oauth,omitempty"`
	OauthResource             *string                        `json:"oauth_resource,omitempty"`
	OmitToolsFrom             []ToolExposureSurface          `json:"omit_tools_from,omitempty"`
	Required                  *bool                          `json:"required,omitempty"`
	Scopes                    []string                       `json:"scopes,omitempty"`
	StartupTimeoutMS          *int64                         `json:"startup_timeout_ms,omitempty"`
	StartupTimeoutSEC         *float64                       `json:"startup_timeout_sec,omitempty"`
	SupportsParallelToolCalls *bool                          `json:"supports_parallel_tool_calls,omitempty"`
	ToolTimeoutSEC            *float64                       `json:"tool_timeout_sec,omitempty"`
	Tools                     map[string]MCPServerToolConfig `json:"tools,omitempty"`
	URL                       *string                        `json:"url,omitempty"`
}

type MCPServerEnvVarClass struct {
	Name   string  `json:"name"`
	Source *string `json:"source,omitempty"`
}

// OAuth client settings used when Codex launches an MCP OAuth flow.
type MCPServerOAuthConfig struct {
	// Fixed callback port that takes precedence over Codex's global OAuth callback port.
	CallbackPort *int64 `json:"callback_port,omitempty"`
	// Explicit OAuth client identifier to present during authorization and token exchange.
	ClientID *string `json:"client_id,omitempty"`
}

// Per-tool approval settings for a single MCP server tool.
type MCPServerToolConfig struct {
	// Approval mode for this tool.
	ApprovalMode *AppToolApproval `json:"approval_mode,omitempty"`
}

type MarketplaceConfig struct {
	// Git revision Codex last successfully activated for this marketplace.
	LastRevision *string `json:"last_revision,omitempty"`
	// Last time Codex successfully added or refreshed this marketplace.
	LastUpdated *string `json:"last_updated,omitempty"`
	// Git ref to check out when `source_type` is `git`.
	Ref *string `json:"ref,omitempty"`
	// Source location used when the marketplace was added.
	Source *string `json:"source,omitempty"`
	// Source kind used to install this marketplace.
	SourceType *MarketplaceSourceType `json:"source_type,omitempty"`
	// Sparse checkout paths used when `source_type` is `git`.
	SparsePaths []string `json:"sparse_paths,omitempty"`
}

// Memories subsystem settings.
//
// Memories settings loaded from config.toml.
type MemoriesToml struct {
	// Model used for memory consolidation.
	ConsolidationModel *string `json:"consolidation_model,omitempty"`
	// When `true`, expose dedicated memory tools through the extension tool surface.
	DedicatedTools *bool `json:"dedicated_tools,omitempty"`
	// When `true`, external context sources mark the thread `memory_mode` as `"polluted"`.
	DisableOnExternalContext *bool `json:"disable_on_external_context,omitempty"`
	// Model used for thread summarisation.
	ExtractModel *string `json:"extract_model,omitempty"`
	// When `false`, newly created threads are stored with `memory_mode = "disabled"` in the
	// state DB.
	GenerateMemories *bool `json:"generate_memories,omitempty"`
	// Maximum number of recent raw memories retained for global consolidation.
	MaxRawMemoriesForConsolidation *int64 `json:"max_raw_memories_for_consolidation,omitempty"`
	// Maximum age of the threads used for memories.
	MaxRolloutAgeDays *int64 `json:"max_rollout_age_days,omitempty"`
	// Maximum number of rollout candidates processed per pass.
	MaxRolloutsPerStartup *int64 `json:"max_rollouts_per_startup,omitempty"`
	// Maximum number of days since a memory was last used before it becomes ineligible for
	// phase 2 selection.
	MaxUnusedDays *int64 `json:"max_unused_days,omitempty"`
	// Minimum remaining percentage required in Codex rate-limit windows before memory startup
	// runs.
	MinRateLimitRemainingPercent *int64 `json:"min_rate_limit_remaining_percent,omitempty"`
	// Minimum idle time between last thread activity and memory creation (hours). > 12h
	// recommended.
	MinRolloutIdleHours *int64 `json:"min_rollout_idle_hours,omitempty"`
	// When `false`, skip injecting memory usage instructions into developer prompts.
	UseMemories *bool `json:"use_memories,omitempty"`
}

// Serializable representation of a provider definition.
type ModelProviderInfo struct {
	// Command-backed bearer-token configuration for this provider.
	Auth *ModelProviderAuthInfo `json:"auth,omitempty"`
	// AWS SigV4 auth configuration for this provider.
	Aws *ModelProviderAwsAuthInfo `json:"aws,omitempty"`
	// Base URL for the provider's OpenAI-compatible API.
	BaseURL *string `json:"base_url,omitempty"`
	// Optional HTTP headers to include in requests to this provider where the (key, value)
	// pairs are the header name and _environment variable_ whose value should be used. If the
	// environment variable is not set, or the value is empty, the header will not be included
	// in the request.
	EnvHTTPHeaders map[string]string `json:"env_http_headers,omitempty"`
	// Environment variable that stores the user's API key for this provider.
	EnvKey *string `json:"env_key,omitempty"`
	// Optional instructions to help the user get a valid value for the variable and set it.
	EnvKeyInstructions *string `json:"env_key_instructions,omitempty"`
	// Value to use with `Authorization: Bearer <token>` header. Use of this config is
	// discouraged in favor of `env_key` for security reasons, but this may be necessary when
	// using this programmatically.
	ExperimentalBearerToken *string `json:"experimental_bearer_token,omitempty"`
	// Additional HTTP headers to include in requests to this provider where the (key, value)
	// pairs are the header name and value.
	HTTPHeaders map[string]string `json:"http_headers,omitempty"`
	// Friendly display name.
	Name *string `json:"name,omitempty"`
	// Optional query parameters to append to the base URL.
	QueryParams map[string]string `json:"query_params,omitempty"`
	// Maximum number of times to retry a failed HTTP request to this provider.
	RequestMaxRetries *int64 `json:"request_max_retries,omitempty"`
	// Does this provider require an OpenAI API Key or ChatGPT login token? If true, user is
	// presented with login screen on first run, and login preference and token/key are stored
	// in auth.json. If false (which is the default), login screen is skipped, and API key (if
	// needed) comes from the "env_key" environment variable.
	RequiresOpenaiAuth *bool `json:"requires_openai_auth,omitempty"`
	// Idle timeout (in milliseconds) to wait for activity on a streaming response before
	// treating the connection as lost.
	StreamIdleTimeoutMS *int64 `json:"stream_idle_timeout_ms,omitempty"`
	// Number of times to retry reconnecting a dropped streaming response before failing.
	StreamMaxRetries *int64 `json:"stream_max_retries,omitempty"`
	// Whether this provider supports the standalone web-search endpoint.
	SupportsStandaloneWebSearch *bool `json:"supports_standalone_web_search,omitempty"`
	// Whether this provider supports the Responses API WebSocket transport.
	SupportsWebsockets *bool `json:"supports_websockets,omitempty"`
	// Maximum time (in milliseconds) to wait for a websocket connection attempt before treating
	// it as failed.
	WebsocketConnectTimeoutMS *int64 `json:"websocket_connect_timeout_ms,omitempty"`
	// Which wire protocol this provider expects.
	WireAPI *WireAPI `json:"wire_api,omitempty"`
}

// Command-backed bearer-token configuration for this provider.
//
// Configuration for obtaining a provider bearer token from a command.
type ModelProviderAuthInfo struct {
	// Command arguments.
	Args []string `json:"args,omitempty"`
	// Command to execute. Bare names are resolved via `PATH`; paths are resolved against `cwd`.
	Command string `json:"command"`
	// Working directory used when running the token command.
	Cwd *string `json:"cwd,omitempty"`
	// Maximum age for the cached token before rerunning the command. Set to `0` to disable
	// proactive refresh and only rerun after a 401 retry path.
	RefreshIntervalMS *int64 `json:"refresh_interval_ms,omitempty"`
	// Maximum time to wait for the token command to exit successfully.
	TimeoutMS *int64 `json:"timeout_ms,omitempty"`
}

// AWS SigV4 auth configuration for this provider.
//
// AWS SigV4 auth configuration for a model provider.
type ModelProviderAwsAuthInfo struct {
	// Optional command used to reauthenticate after a refreshable AWS auth failure.
	AuthRefresh *AwsAuthRefreshConfig `json:"auth_refresh,omitempty"`
	// AWS profile name to use. When unset, the AWS SDK default chain decides.
	Profile *string `json:"profile,omitempty"`
	// AWS region to use for provider-specific endpoints.
	Region *string `json:"region,omitempty"`
}

// Optional command used to reauthenticate after a refreshable AWS auth failure.
//
// Command used to refresh AWS credentials for a model provider.
type AwsAuthRefreshConfig struct {
	// Arguments passed to the refresh command.
	Args []string `json:"args,omitempty"`
	// Executable to invoke directly, without a shell.
	Command string `json:"command"`
	// Maximum time to wait for the refresh command to complete.
	TimeoutMS *int64 `json:"timeout_ms,omitempty"`
}

// Collection of in-product notices (different from notifications) See
// [`crate::types::Notice`] for more details
type Notice struct {
	// Tracks scopes where external config migration prompts should be suppressed.
	ExternalConfigMigrationPrompts *ExternalConfigMigrationPrompts `json:"external_config_migration_prompts,omitempty"`
	// Tracks whether the user opted out of Codex-managed fast defaults.
	FastDefaultOptOut *bool `json:"fast_default_opt_out,omitempty"`
	// Tracks whether the user has acknowledged the full access warning prompt.
	HideFullAccessWarning *bool `json:"hide_full_access_warning,omitempty"`
	// Tracks whether the user has seen the gpt-5.1-codex-max migration prompt
	HideGPT51CodexMaxMigrationPrompt *bool `json:"hide_gpt-5.1-codex-max_migration_prompt,omitempty"`
	// Tracks whether the user has seen the model migration prompt
	HideGpt51_MigrationPrompt *bool `json:"hide_gpt5_1_migration_prompt,omitempty"`
	// Tracks whether the user opted out of the rate limit model switch reminder.
	HideRateLimitModelNudge *bool `json:"hide_rate_limit_model_nudge,omitempty"`
	// Tracks whether the user has acknowledged the Windows world-writable directories warning.
	HideWorldWritableWarning *bool `json:"hide_world_writable_warning,omitempty"`
	// Tracks acknowledged model migrations as old->new model slug mappings.
	ModelMigrations map[string]string `json:"model_migrations,omitempty"`
}

// Tracks scopes where external config migration prompts should be suppressed.
//
// Settings for notices we display to users via the tui and app-server clients (primarily
// the Codex IDE extension). NOTE: these are different from notifications - notices are
// warnings, NUX screens, acknowledgements, etc.
type ExternalConfigMigrationPrompts struct {
	// Tracks whether home-level external config migration prompts are hidden.
	Home *bool `json:"home,omitempty"`
	// Tracks the last time the home-level external config migration prompt was shown.
	HomeLastPromptedAt *int64 `json:"home_last_prompted_at,omitempty"`
	// Tracks the last time a project-level external config migration prompt was shown.
	ProjectLastPromptedAt map[string]int64 `json:"project_last_prompted_at,omitempty"`
	// Tracks which project paths have opted out of external config migration prompts.
	Projects map[string]bool `json:"projects,omitempty"`
}

// Orchestrator-owned feature settings.
type OrchestratorToml struct {
	MCP    *OrchestratorFeatureToml `json:"mcp,omitempty"`
	Skills *OrchestratorFeatureToml `json:"skills,omitempty"`
}

// Settings for a feature owned by the orchestrator.
type OrchestratorFeatureToml struct {
	Enabled *bool `json:"enabled,omitempty"`
}

// OTEL configuration.
//
// OTEL settings loaded from config.toml. Fields are optional so we can apply defaults.
type OtelConfigToml struct {
	// Mark traces with environment (dev, staging, prod, test). Defaults to dev.
	Environment *string `json:"environment,omitempty"`
	// Optional log exporter
	Exporter *OtelExporterKind `json:"exporter"`
	// Log user prompt in traces
	LogUserPrompt *bool `json:"log_user_prompt,omitempty"`
	// Optional metrics exporter
	MetricsExporter *OtelExporterKind `json:"metrics_exporter"`
	// Attributes to add to every exported trace span.
	SpanAttributes map[string]string `json:"span_attributes,omitempty"`
	// Byte limit for tool-result log output; independent of model-visible output.
	ToolResult *ToolResultLogConfig `json:"tool_result,omitempty"`
	// Optional trace exporter
	TraceExporter *OtelExporterKind `json:"trace_exporter"`
	// Semicolon-separated `key:value` fields to upsert into W3C tracestate members.
	Tracestate map[string]map[string]string `json:"tracestate,omitempty"`
}

type OtelExporterKindClass struct {
	OtlpHTTP *OtlpHTTP `json:"otlp-http,omitempty"`
	OtlpGrpc *OtlpGrpc `json:"otlp-grpc,omitempty"`
}

type OtlpGrpc struct {
	Endpoint string            `json:"endpoint"`
	Headers  map[string]string `json:"headers,omitempty"`
	TLS      *OtelTLSConfig    `json:"tls,omitempty"`
}

type OtelTLSConfig struct {
	CACertificate     *string `json:"ca-certificate,omitempty"`
	ClientCertificate *string `json:"client-certificate,omitempty"`
	ClientPrivateKey  *string `json:"client-private-key,omitempty"`
}

type OtlpHTTP struct {
	Endpoint string            `json:"endpoint"`
	Headers  map[string]string `json:"headers,omitempty"`
	Protocol OtelHTTPProtocol  `json:"protocol"`
	TLS      *OtelTLSConfig    `json:"tls,omitempty"`
}

// Byte limit for tool-result log output; independent of model-visible output.
//
// Limit for the text included in `codex.tool_result` log records. This does not affect
// model-visible output. Raising it can expose more tool data to logs.
type ToolResultLogConfig struct {
	// Maximum UTF-8 bytes before the truncation notice. Defaults to 2048.
	MaxBytes *int64 `json:"max_bytes,omitempty"`
}

type PluginConfig struct {
	Enabled *bool `json:"enabled,omitempty"`
	// Per-MCP-server policy overlays for MCP servers contributed by this plugin.
	MCPServers map[string]PluginMCPServerConfig `json:"mcp_servers,omitempty"`
}

// Policy settings for a plugin-provided MCP server.
//
// This intentionally excludes transport settings: plugin manifests own how the MCP server
// is launched, while user config owns enablement and tool policy.
type PluginMCPServerConfig struct {
	// Approval mode for tools in this server unless a tool override exists.
	DefaultToolsApprovalMode *AppToolApproval `json:"default_tools_approval_mode,omitempty"`
	// Explicit deny-list of tools. These tools are removed after applying `enabled_tools`.
	DisabledTools []string `json:"disabled_tools,omitempty"`
	// When `false`, Codex skips initializing this plugin MCP server.
	Enabled *bool `json:"enabled,omitempty"`
	// Explicit allow-list of tools exposed from this server.
	EnabledTools []string `json:"enabled_tools,omitempty"`
	// Per-tool approval settings keyed by tool name.
	Tools map[string]MCPServerToolConfig `json:"tools,omitempty"`
}

// Collection of common configuration options that a user can define as a unit in
// `config.toml`.
type ConfigProfile struct {
	Analytics                      *AnalyticsConfigToml `json:"analytics,omitempty"`
	ApprovalPolicy                 *AskForApproval      `json:"approval_policy"`
	ApprovalsReviewer              *ApprovalsReviewer   `json:"approvals_reviewer,omitempty"`
	ChatgptBaseURL                 *string              `json:"chatgpt_base_url,omitempty"`
	ExperimentalCompactPromptFile  *string              `json:"experimental_compact_prompt_file,omitempty"`
	ExperimentalUseUnifiedExecTool *bool                `json:"experimental_use_unified_exec_tool,omitempty"`
	// Optional feature toggles scoped to this profile.
	Features                             *ProfileFeatures `json:"features,omitempty"`
	IncludeAppsInstructions              *bool            `json:"include_apps_instructions,omitempty"`
	IncludeCollaborationModeInstructions *bool            `json:"include_collaboration_mode_instructions,omitempty"`
	IncludeEnvironmentContext            *bool            `json:"include_environment_context,omitempty"`
	IncludePermissionsInstructions       *bool            `json:"include_permissions_instructions,omitempty"`
	Model                                *string          `json:"model,omitempty"`
	// Optional path to a JSON model catalog (applied on startup only).
	ModelCatalogJSON *string `json:"model_catalog_json,omitempty"`
	// Optional path to a file containing model instructions.
	ModelInstructionsFile *string `json:"model_instructions_file,omitempty"`
	// The key in the `model_providers` map identifying the [`ModelProviderInfo`] to use.
	ModelProvider           *string           `json:"model_provider,omitempty"`
	ModelReasoningEffort    *string           `json:"model_reasoning_effort,omitempty"`
	ModelReasoningSummary   *ReasoningSummary `json:"model_reasoning_summary,omitempty"`
	ModelVerbosity          *Verbosity        `json:"model_verbosity,omitempty"`
	OSSProvider             *string           `json:"oss_provider,omitempty"`
	Personality             *Personality      `json:"personality,omitempty"`
	PlanModeReasoningEffort *string           `json:"plan_mode_reasoning_effort,omitempty"`
	SandboxMode             *SandboxMode      `json:"sandbox_mode,omitempty"`
	// Optional explicit service tier request id for new turns (for example `default`,
	// `priority`, or `flex`; legacy `fast` also works).
	ServiceTier *string    `json:"service_tier,omitempty"`
	Tools       *ToolsToml `json:"tools,omitempty"`
	// TUI settings scoped to this profile.
	Tui       *ProfileTui    `json:"tui,omitempty"`
	WebSearch *WebSearchMode `json:"web_search,omitempty"`
	Windows   *WindowsToml   `json:"windows,omitempty"`
}

// Optional feature toggles scoped to this profile.
type ProfileFeatures struct {
	ApplyPatchFreeform                  *bool                                            `json:"apply_patch_freeform,omitempty"`
	ApplyPatchPreserveLineEndings       *bool                                            `json:"apply_patch_preserve_line_endings,omitempty"`
	ApplyPatchStreamingEvents           *bool                                            `json:"apply_patch_streaming_events,omitempty"`
	Apps                                *bool                                            `json:"apps,omitempty"`
	AppsMCPPathOverride                 *StickyAppsMCPPathOverride                       `json:"apps_mcp_path_override"`
	AuthElicitation                     *bool                                            `json:"auth_elicitation,omitempty"`
	BackgroundPaginatedRolloutMigration *bool                                            `json:"background_paginated_rollout_migration,omitempty"`
	BrowserUse                          *bool                                            `json:"browser_use,omitempty"`
	BrowserUseExternal                  *bool                                            `json:"browser_use_external,omitempty"`
	BrowserUseFullCDPAccess             *bool                                            `json:"browser_use_full_cdp_access,omitempty"`
	Chronicle                           *bool                                            `json:"chronicle,omitempty"`
	CodeMode                            *FeatureTomlForCodeModeConfigToml                `json:"code_mode"`
	CodeModeBufferedExec                *bool                                            `json:"code_mode_buffered_exec,omitempty"`
	CodeModeHost                        *FeatureTomlForCodeModeHostConfigToml            `json:"code_mode_host"`
	CodeModeInterrupt                   *bool                                            `json:"code_mode_interrupt,omitempty"`
	CodeModeOnly                        *bool                                            `json:"code_mode_only,omitempty"`
	CodexGitCommit                      *bool                                            `json:"codex_git_commit,omitempty"`
	CodexHooks                          *bool                                            `json:"codex_hooks,omitempty"`
	Collab                              *bool                                            `json:"collab,omitempty"`
	CollaborationModes                  *bool                                            `json:"collaboration_modes,omitempty"`
	CompactionImageBudget               *bool                                            `json:"compaction_image_budget,omitempty"`
	ComputerUse                         *bool                                            `json:"computer_use,omitempty"`
	ConcurrentReasoningSummaries        *bool                                            `json:"concurrent_reasoning_summaries,omitempty"`
	Connectors                          *bool                                            `json:"connectors,omitempty"`
	ContentItemKinds                    *bool                                            `json:"content_item_kinds,omitempty"`
	CurrentTimeReminder                 *FeatureTomlForCurrentTimeReminderConfigToml     `json:"current_time_reminder"`
	CwdRelativeTurnDiffs                *bool                                            `json:"cwd_relative_turn_diffs,omitempty"`
	DefaultModeRequestUserInput         *bool                                            `json:"default_mode_request_user_input,omitempty"`
	DeferredExecutor                    *bool                                            `json:"deferred_executor,omitempty"`
	DeferredToolWorldState              *bool                                            `json:"deferred_tool_world_state,omitempty"`
	ElevatedWindowsSandbox              *bool                                            `json:"elevated_windows_sandbox,omitempty"`
	EnableExperimentalWindowsSandbox    *bool                                            `json:"enable_experimental_windows_sandbox,omitempty"`
	EnableFanout                        *bool                                            `json:"enable_fanout,omitempty"`
	EnableMCPApps                       *bool                                            `json:"enable_mcp_apps,omitempty"`
	EnableRequestCompression            *bool                                            `json:"enable_request_compression,omitempty"`
	ExecPermissionApprovals             *bool                                            `json:"exec_permission_approvals,omitempty"`
	ExecutedToolCallMetadata            *bool                                            `json:"executed_tool_call_metadata,omitempty"`
	ExecutorCapabilityDiscovery         *bool                                            `json:"executor_capability_discovery,omitempty"`
	ExperimentalUseUnifiedExecTool      *bool                                            `json:"experimental_use_unified_exec_tool,omitempty"`
	ExperimentalWindowsSandbox          *bool                                            `json:"experimental_windows_sandbox,omitempty"`
	ExternalAgentMemoryImport           *bool                                            `json:"external_agent_memory_import,omitempty"`
	ExternalMigration                   *bool                                            `json:"external_migration,omitempty"`
	FastMode                            *bool                                            `json:"fast_mode,omitempty"`
	Goals                               *bool                                            `json:"goals,omitempty"`
	GuardianApproval                    *bool                                            `json:"guardian_approval,omitempty"`
	GuardianEnhancedNodeReplTranscripts *bool                                            `json:"guardian_enhanced_node_repl_transcripts,omitempty"`
	GuardianEXT                         *bool                                            `json:"guardian_ext,omitempty"`
	GuardianNodeReplTranscriptImages    *bool                                            `json:"guardian_node_repl_transcript_images,omitempty"`
	GuardianReuseParentCompaction       *bool                                            `json:"guardian_reuse_parent_compaction,omitempty"`
	Guardianv2                          *FeatureTomlForGuardianV2ConfigToml              `json:"guardianv2"`
	Hooks                               *bool                                            `json:"hooks,omitempty"`
	ImageDetailOriginal                 *bool                                            `json:"image_detail_original,omitempty"`
	ImageGeneration                     *bool                                            `json:"image_generation,omitempty"`
	ImageResizeNotice                   *bool                                            `json:"image_resize_notice,omitempty"`
	Imagegenext                         *bool                                            `json:"imagegenext,omitempty"`
	InAppBrowser                        *bool                                            `json:"in_app_browser,omitempty"`
	InAppChat                           *bool                                            `json:"in_app_chat,omitempty"`
	InAppDictation                      *bool                                            `json:"in_app_dictation,omitempty"`
	InAppLocalAutomation                *bool                                            `json:"in_app_local_automation,omitempty"`
	InAppUpdates                        *bool                                            `json:"in_app_updates,omitempty"`
	ItemIDS                             *bool                                            `json:"item_ids,omitempty"`
	JSRepl                              *bool                                            `json:"js_repl,omitempty"`
	JSReplToolsOnly                     *bool                                            `json:"js_repl_tools_only,omitempty"`
	LocalThreadStoreCompression         *bool                                            `json:"local_thread_store_compression,omitempty"`
	MCP2026_07_28                       *bool                                            `json:"mcp_2026_07_28,omitempty"`
	Memories                            *bool                                            `json:"memories,omitempty"`
	MemoryTool                          *bool                                            `json:"memory_tool,omitempty"`
	MentionsV2                          *bool                                            `json:"mentions_v2,omitempty"`
	MultiAgent                          *bool                                            `json:"multi_agent,omitempty"`
	MultiAgentMode                      *bool                                            `json:"multi_agent_mode,omitempty"`
	MultiAgentV2                        *FeatureTomlForMultiAgentV2ConfigToml            `json:"multi_agent_v2"`
	NetworkProxy                        *FeatureTomlForNetworkProxyConfigToml            `json:"network_proxy"`
	NonPrefixedMCPToolNames             *FeatureTomlForNonPrefixedMCPToolNamesConfigToml `json:"non_prefixed_mcp_tool_names"`
	Personality                         *bool                                            `json:"personality,omitempty"`
	PluginHooks                         *bool                                            `json:"plugin_hooks,omitempty"`
	PluginSharing                       *bool                                            `json:"plugin_sharing,omitempty"`
	Plugins                             *bool                                            `json:"plugins,omitempty"`
	PreventIdleSleep                    *bool                                            `json:"prevent_idle_sleep,omitempty"`
	Psp                                 *bool                                            `json:"psp,omitempty"`
	RealtimeConversation                *bool                                            `json:"realtime_conversation,omitempty"`
	RecommendedPlugins                  *bool                                            `json:"recommended_plugins,omitempty"`
	RemoteCompactionV2                  *bool                                            `json:"remote_compaction_v2,omitempty"`
	RemoteControl                       *bool                                            `json:"remote_control,omitempty"`
	RemoteModels                        *bool                                            `json:"remote_models,omitempty"`
	RemotePlugin                        *bool                                            `json:"remote_plugin,omitempty"`
	RequestPermissions                  *bool                                            `json:"request_permissions,omitempty"`
	RequestPermissionsTool              *bool                                            `json:"request_permissions_tool,omitempty"`
	RequestRule                         *bool                                            `json:"request_rule,omitempty"`
	ResizeAllImages                     *bool                                            `json:"resize_all_images,omitempty"`
	RespectSystemProxy                  *bool                                            `json:"respect_system_proxy,omitempty"`
	ResponsesWebsockets                 *bool                                            `json:"responses_websockets,omitempty"`
	ResponsesWebsocketsV2               *bool                                            `json:"responses_websockets_v2,omitempty"`
	RetainClientDeveloperMessages       *bool                                            `json:"retain_client_developer_messages,omitempty"`
	RolloutBudget                       *FeatureTomlForRolloutBudgetConfigToml           `json:"rollout_budget"`
	RuntimeMetrics                      *bool                                            `json:"runtime_metrics,omitempty"`
	SearchTool                          *bool                                            `json:"search_tool,omitempty"`
	SecretAuthStorage                   *bool                                            `json:"secret_auth_storage,omitempty"`
	SendAsyncMessage                    *bool                                            `json:"send_async_message,omitempty"`
	ShellSnapshot                       *bool                                            `json:"shell_snapshot,omitempty"`
	ShellSnapshotV2                     *bool                                            `json:"shell_snapshot_v2,omitempty"`
	ShellTool                           *bool                                            `json:"shell_tool,omitempty"`
	ShellZshFork                        *bool                                            `json:"shell_zsh_fork,omitempty"`
	SkillEnvVarDependencyPrompt         *bool                                            `json:"skill_env_var_dependency_prompt,omitempty"`
	SkillMCPDependencyInstall           *bool                                            `json:"skill_mcp_dependency_install,omitempty"`
	SkillSearch                         *bool                                            `json:"skill_search,omitempty"`
	Sqlite                              *bool                                            `json:"sqlite,omitempty"`
	StandaloneWebSearch                 *bool                                            `json:"standalone_web_search,omitempty"`
	Steer                               *bool                                            `json:"steer,omitempty"`
	Telepathy                           *bool                                            `json:"telepathy,omitempty"`
	TerminalResizeReflow                *bool                                            `json:"terminal_resize_reflow,omitempty"`
	TerminalVisualizationInstructions   *bool                                            `json:"terminal_visualization_instructions,omitempty"`
	TokenBudget                         *FeatureTomlForTokenBudgetConfigToml             `json:"token_budget"`
	ToolCallMCPElicitation              *bool                                            `json:"tool_call_mcp_elicitation,omitempty"`
	ToolRegistry                        *ToolRegistryConfigToml                          `json:"tool_registry,omitempty"`
	ToolSearch                          *bool                                            `json:"tool_search,omitempty"`
	ToolSearchAlwaysDeferMCPTools       *bool                                            `json:"tool_search_always_defer_mcp_tools,omitempty"`
	ToolSuggest                         *bool                                            `json:"tool_suggest,omitempty"`
	TuiAppServer                        *bool                                            `json:"tui_app_server,omitempty"`
	UnavailableDummyTools               *bool                                            `json:"unavailable_dummy_tools,omitempty"`
	UnboundedConnectionRetries          *bool                                            `json:"unbounded_connection_retries,omitempty"`
	Undo                                *bool                                            `json:"undo,omitempty"`
	UnifiedExec                         *bool                                            `json:"unified_exec,omitempty"`
	UnifiedExecZshFork                  *bool                                            `json:"unified_exec_zsh_fork,omitempty"`
	UnifiedImageBudget                  *bool                                            `json:"unified_image_budget,omitempty"`
	UseAgentIdentity                    *bool                                            `json:"use_agent_identity,omitempty"`
	UseLegacyLandlock                   *bool                                            `json:"use_legacy_landlock,omitempty"`
	UseLinuxSandboxBwrap                *bool                                            `json:"use_linux_sandbox_bwrap,omitempty"`
	ViewImage                           *bool                                            `json:"view_image,omitempty"`
	WebSearch                           *bool                                            `json:"web_search,omitempty"`
	WebSearchCached                     *bool                                            `json:"web_search_cached,omitempty"`
	WebSearchRequest                    *bool                                            `json:"web_search_request,omitempty"`
	WorkspaceDependencies               *bool                                            `json:"workspace_dependencies,omitempty"`
	WorkspaceOwnerUsageNudge            *bool                                            `json:"workspace_owner_usage_nudge,omitempty"`
}

type FluffyAppsMCPPathOverride struct {
	Enabled *bool   `json:"enabled,omitempty"`
	Path    *string `json:"path,omitempty"`
}

// Nested tools section for feature toggles
type ToolsToml struct {
	ExperimentalRequestUserInput *ExperimentalRequestUserInput `json:"experimental_request_user_input,omitempty"`
	UpdatePlan                   *UpdatePlanToolConfig         `json:"update_plan,omitempty"`
	WebSearch                    *WebSearchToolConfig          `json:"web_search,omitempty"`
}

type ExperimentalRequestUserInput struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type UpdatePlanToolConfig struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type WebSearchToolConfig struct {
	AllowedDomains []string           `json:"allowed_domains,omitempty"`
	ContextSize    *Verbosity         `json:"context_size,omitempty"`
	Location       *WebSearchLocation `json:"location,omitempty"`
}

type WebSearchLocation struct {
	City     *string `json:"city,omitempty"`
	Country  *string `json:"country,omitempty"`
	Region   *string `json:"region,omitempty"`
	Timezone *string `json:"timezone,omitempty"`
}

// TUI settings scoped to this profile.
//
// TUI settings supported inside a named profile.
type ProfileTui struct {
	// Preferred layout for resume/fork session picker results.
	SessionPickerView *SessionPickerViewMode `json:"session_picker_view,omitempty"`
}

// Windows-specific configuration.
type WindowsToml struct {
	Sandbox *WindowsSandboxModeToml `json:"sandbox,omitempty"`
	// Defaults to `true`. Set to `false` to launch the final sandboxed child process on
	// `Winsta0\\Default` instead of a private desktop.
	SandboxPrivateDesktop *bool `json:"sandbox_private_desktop,omitempty"`
}

type ProjectConfig struct {
	TrustLevel *TrustLevel `json:"trust_level,omitempty"`
}

// Experimental / do not use. Realtime websocket session selection. `version` controls v1/v2
// and `type` controls conversational/transcription.
type RealtimeToml struct {
	Transport *RealtimeTransport           `json:"transport,omitempty"`
	Type      *RealtimeWsMode              `json:"type,omitempty"`
	Version   *RealtimeConversationVersion `json:"version,omitempty"`
	Voice     *RealtimeVoice               `json:"voice,omitempty"`
}

// Sandbox configuration to apply if `sandbox` is `WorkspaceWrite`.
type SandboxWorkspaceWrite struct {
	ExcludeSlashTmp     *bool    `json:"exclude_slash_tmp,omitempty"`
	ExcludeTmpdirEnvVar *bool    `json:"exclude_tmpdir_env_var,omitempty"`
	NetworkAccess       *bool    `json:"network_access,omitempty"`
	WritableRoots       []string `json:"writable_roots,omitempty"`
}

// Policy for building the `env` when spawning a process via shell-like tools.
type ShellEnvironmentPolicyToml struct {
	// Legacy list of regular expressions to exclude.
	Exclude                []string `json:"exclude,omitempty"`
	ExperimentalUseProfile *bool    `json:"experimental_use_profile,omitempty"`
	// Pattern actions used by the canonical table representation.
	//
	// Ordinary config keeps accepting the legacy arrays above during the migration.
	// Requirements will accept only this keyed form, keeping array compatibility isolated so
	// the legacy fields can be deprecated later. Pattern keys merge case-insensitively across
	// config layers, matching how the resulting patterns match environment variable names.
	Filters               map[string]ShellEnvironmentPolicyFilter `json:"filters,omitempty"`
	IgnoreDefaultExcludes *bool                                   `json:"ignore_default_excludes,omitempty"`
	// Legacy list of regular expressions to include.
	IncludeOnly []string                       `json:"include_only,omitempty"`
	Inherit     *ShellEnvironmentPolicyInherit `json:"inherit,omitempty"`
	Set         map[string]string              `json:"set,omitempty"`
}

// User-level skill config entries keyed by SKILL.md path.
type SkillsConfig struct {
	Bundled *BundledSkillsConfig `json:"bundled,omitempty"`
	Config  []SkillConfig        `json:"config,omitempty"`
	// Whether turns receive the automatic skills instructions block.
	IncludeInstructions *bool `json:"include_instructions,omitempty"`
	// Maximum tokens used by the available-skills catalog. Defaults to 2% of the model context
	// window and is capped at 10,000 tokens when set.
	MaxContextTokens *int64 `json:"max_context_tokens,omitempty"`
}

type BundledSkillsConfig struct {
	Enabled *bool `json:"enabled,omitempty"`
}

type SkillConfig struct {
	Enabled bool `json:"enabled"`
	// Name-based selector.
	Name *string `json:"name,omitempty"`
	// Path-based selector.
	Path *string `json:"path,omitempty"`
}

// Additional discoverable tools that can be suggested for installation.
type ToolSuggestConfig struct {
	DisabledTools []ToolSuggestDisabledTool `json:"disabled_tools,omitempty"`
	Discoverables []ToolSuggestDiscoverable `json:"discoverables,omitempty"`
}

type ToolSuggestDisabledTool struct {
	ID   string                      `json:"id"`
	Type ToolSuggestDiscoverableType `json:"type"`
}

type ToolSuggestDiscoverable struct {
	ID   string                      `json:"id"`
	Type ToolSuggestDiscoverableType `json:"type"`
}

// Collection of settings that are specific to the TUI.
type Tui struct {
	// Controls whether the TUI uses the terminal's alternate screen buffer.
	//
	// - `auto` (default): Use alternate screen. - `always`: Always use alternate screen. -
	// `never`: Never use alternate screen (inline mode only, preserves scrollback).
	AlternateScreen *AltScreenMode `json:"alternate_screen,omitempty"`
	// Enable animations (welcome screen, shimmer effects, spinners). Defaults to `true`.
	Animations *bool `json:"animations,omitempty"`
	// Keybinding overrides for the TUI.
	//
	// This supports rebinding selected actions globally and by context. Context bindings take
	// precedence over `global` bindings.
	Keymap *TuiKeymap `json:"keymap,omitempty"`
	// Startup tooltip availability NUX state persisted by the TUI.
	ModelAvailabilityNux map[string]int64 `json:"model_availability_nux,omitempty"`
	// Controls whether TUI notifications are delivered only when the terminal is unfocused or
	// regardless of focus. Defaults to `unfocused`.
	NotificationCondition *NotificationCondition `json:"notification_condition,omitempty"`
	// Notification method to use for terminal notifications. Defaults to `auto`.
	NotificationMethod *NotificationMethod `json:"notification_method,omitempty"`
	// Enable desktop notifications from the TUI. Defaults to `true`.
	Notifications *Notifications `json:"notifications"`
	// Pet id to preselect in the terminal pet picker.
	//
	// Custom pet ids resolve against CODEX_HOME/pets/<pet-id>/pet.json.
	Pet *string `json:"pet,omitempty"`
	// Where the terminal pet should anchor vertically.
	//
	// Defaults to `composer`, which follows the current TUI composer viewport.
	PetAnchor *TuiPetAnchor `json:"pet_anchor,omitempty"`
	// Start the TUI in raw scrollback mode for copy-friendly transcript output. Defaults to
	// `false`.
	RawOutputMode *bool `json:"raw_output_mode,omitempty"`
	// Working directory to use when resuming or forking a session. When unset, prompt if the
	// current and session directories differ.
	ResumeCwd *ResumeCwdMode `json:"resume_cwd,omitempty"`
	// Preferred layout for resume/fork session picker results.
	SessionPickerView *SessionPickerViewMode `json:"session_picker_view,omitempty"`
	// Show startup tooltips in the TUI welcome screen. Defaults to `true`.
	ShowTooltips *bool `json:"show_tooltips,omitempty"`
	// Ordered list of status line item identifiers.
	//
	// When set, the TUI renders the selected items as the status line. When unset, the TUI
	// defaults to: `model-with-reasoning` and `current-dir`.
	StatusLine []string `json:"status_line,omitempty"`
	// Color status line items with colors derived from the active syntax theme. Defaults to
	// `true`.
	StatusLineUseColors *bool `json:"status_line_use_colors,omitempty"`
	// Trim terminal resize-reflow replay to the most recent rendered terminal rows when the
	// transcript exceeds this cap. Omit to use Codex's terminal-specific default. Set to `0` to
	// keep all rendered rows.
	TerminalResizeReflowMaxRows *int64 `json:"terminal_resize_reflow_max_rows,omitempty"`
	// Ordered list of terminal title item identifiers.
	//
	// When set, the TUI renders the selected items into the terminal window/tab title. When
	// unset, the TUI defaults to: `activity` and `project`. The `activity` item spins while
	// working and shows an action-required message when blocked on the user.
	TerminalTitle []string `json:"terminal_title,omitempty"`
	// Syntax highlighting theme name (kebab-case).
	//
	// When set, overrides automatic light/dark theme detection. Use `/theme` in the TUI or see
	// `$CODEX_HOME/themes` for custom themes.
	Theme *string `json:"theme,omitempty"`
	// Start the composer in Vim mode (`Normal`) by default. Defaults to `false`.
	VimModeDefault *bool `json:"vim_mode_default,omitempty"`
}

// Keybinding overrides for the TUI.
//
// This supports rebinding selected actions globally and by context. Context bindings take
// precedence over `global` bindings.
//
// Raw keymap configuration from `[tui.keymap]`.
//
// Each context contains action-level overrides. Missing actions inherit from built-in
// defaults, and selected chat/composer actions can fall back through `global` during
// runtime resolution.
//
// This type is intentionally a persistence shape, not the structure used by input handlers.
// Runtime consumers should resolve it into `RuntimeKeymap` first so precedence, empty-list
// unbinding, and duplicate-key validation are applied consistently.
type TuiKeymap struct {
	Agents        *TuiAgentsKeymap        `json:"agents,omitempty"`
	Approval      *TuiApprovalKeymap      `json:"approval,omitempty"`
	Chat          *TuiChatKeymap          `json:"chat,omitempty"`
	Composer      *TuiComposerKeymap      `json:"composer,omitempty"`
	Editor        *TuiEditorKeymap        `json:"editor,omitempty"`
	Global        *TuiGlobalKeymap        `json:"global,omitempty"`
	List          *TuiListKeymap          `json:"list,omitempty"`
	Pager         *TuiPagerKeymap         `json:"pager,omitempty"`
	VimNormal     *TuiVimNormalKeymap     `json:"vim_normal,omitempty"`
	VimOperator   *TuiVimOperatorKeymap   `json:"vim_operator,omitempty"`
	VimTextObject *TuiVimTextObjectKeymap `json:"vim_text_object,omitempty"`
}

// Shortcuts specific to the shared agents overview.
type TuiAgentsKeymap struct {
	// Start composing a new agent task.
	NewTask *ForcedChatgptWorkspaceIDS `json:"new_task"`
	// Rename the selected task.
	Rename *ForcedChatgptWorkspaceIDS `json:"rename"`
	// Search the available agent tasks.
	Search *ForcedChatgptWorkspaceIDS `json:"search"`
	// Stop the selected running task.
	Stop *ForcedChatgptWorkspaceIDS `json:"stop"`
	// Toggle grouping tasks by status or project.
	ToggleGrouping *ForcedChatgptWorkspaceIDS `json:"toggle_grouping"`
}

// Approval overlay keybindings.
type TuiApprovalKeymap struct {
	// Approve the primary option.
	Approve *ForcedChatgptWorkspaceIDS `json:"approve"`
	// Approve with exec-policy prefix when that option exists.
	ApproveForPrefix *ForcedChatgptWorkspaceIDS `json:"approve_for_prefix"`
	// Approve for session when that option exists.
	ApproveForSession *ForcedChatgptWorkspaceIDS `json:"approve_for_session"`
	// Cancel an elicitation request.
	Cancel *ForcedChatgptWorkspaceIDS `json:"cancel"`
	// Decline and provide corrective guidance.
	Decline *ForcedChatgptWorkspaceIDS `json:"decline"`
	// Deny without providing follow-up guidance.
	Deny *ForcedChatgptWorkspaceIDS `json:"deny"`
	// Open the full-screen approval details view.
	OpenFullscreen *ForcedChatgptWorkspaceIDS `json:"open_fullscreen"`
	// Open the thread that requested approval when shown from another thread.
	OpenThread *ForcedChatgptWorkspaceIDS `json:"open_thread"`
}

// Chat context keybindings.
type TuiChatKeymap struct {
	// Decrease the active reasoning effort.
	DecreaseReasoningEffort *ForcedChatgptWorkspaceIDS `json:"decrease_reasoning_effort"`
	// Edit the most recently queued message.
	EditQueuedMessage *ForcedChatgptWorkspaceIDS `json:"edit_queued_message"`
	// Increase the active reasoning effort.
	IncreaseReasoningEffort *ForcedChatgptWorkspaceIDS `json:"increase_reasoning_effort"`
	// Interrupt the active turn.
	InterruptTurn *ForcedChatgptWorkspaceIDS `json:"interrupt_turn"`
	// Switch to the next available permission mode.
	NextPermissionMode *ForcedChatgptWorkspaceIDS `json:"next_permission_mode"`
	// Switch to the previous available permission mode.
	PreviousPermissionMode *ForcedChatgptWorkspaceIDS `json:"previous_permission_mode"`
	// Toggle the microphone in an active voice conversation.
	ToggleVoiceMute *ForcedChatgptWorkspaceIDS `json:"toggle_voice_mute"`
}

// Composer context keybindings. These override corresponding `global` actions.
type TuiComposerKeymap struct {
	// Move to the next match in reverse history search.
	HistorySearchNext *ForcedChatgptWorkspaceIDS `json:"history_search_next"`
	// Open reverse history search or move to the previous match.
	HistorySearchPrevious *ForcedChatgptWorkspaceIDS `json:"history_search_previous"`
	// Queue the current composer draft while a task is running.
	Queue *ForcedChatgptWorkspaceIDS `json:"queue"`
	// Submit the current composer draft.
	Submit *ForcedChatgptWorkspaceIDS `json:"submit"`
	// Toggle the composer shortcut overlay.
	ToggleShortcuts *ForcedChatgptWorkspaceIDS `json:"toggle_shortcuts"`
}

// Editor context keybindings for text editing inside text areas.
type TuiEditorKeymap struct {
	// Delete one grapheme to the left.
	DeleteBackward *ForcedChatgptWorkspaceIDS `json:"delete_backward"`
	// Delete the previous word.
	DeleteBackwardWord *ForcedChatgptWorkspaceIDS `json:"delete_backward_word"`
	// Delete one grapheme to the right.
	DeleteForward *ForcedChatgptWorkspaceIDS `json:"delete_forward"`
	// Delete the next word.
	DeleteForwardWord *ForcedChatgptWorkspaceIDS `json:"delete_forward_word"`
	// Insert a newline in the editor.
	InsertNewline *ForcedChatgptWorkspaceIDS `json:"insert_newline"`
	// Kill text from cursor to line end.
	KillLineEnd *ForcedChatgptWorkspaceIDS `json:"kill_line_end"`
	// Kill text from cursor to line start.
	KillLineStart *ForcedChatgptWorkspaceIDS `json:"kill_line_start"`
	// Kill the current line.
	KillWholeLine *ForcedChatgptWorkspaceIDS `json:"kill_whole_line"`
	// Move cursor down one visual line.
	MoveDown *ForcedChatgptWorkspaceIDS `json:"move_down"`
	// Move cursor left by one grapheme.
	MoveLeft *ForcedChatgptWorkspaceIDS `json:"move_left"`
	// Move cursor to end of line.
	MoveLineEnd *ForcedChatgptWorkspaceIDS `json:"move_line_end"`
	// Move cursor to beginning of line.
	MoveLineStart *ForcedChatgptWorkspaceIDS `json:"move_line_start"`
	// Move cursor right by one grapheme.
	MoveRight *ForcedChatgptWorkspaceIDS `json:"move_right"`
	// Move cursor up one visual line.
	MoveUp *ForcedChatgptWorkspaceIDS `json:"move_up"`
	// Move cursor to beginning of previous word.
	MoveWordLeft *ForcedChatgptWorkspaceIDS `json:"move_word_left"`
	// Move cursor to end of next word.
	MoveWordRight *ForcedChatgptWorkspaceIDS `json:"move_word_right"`
	// Yank the kill buffer.
	Yank *ForcedChatgptWorkspaceIDS `json:"yank"`
}

// Global keybindings. These are used when a context does not define an override.
type TuiGlobalKeymap struct {
	// Clear the terminal UI.
	ClearTerminal *ForcedChatgptWorkspaceIDS `json:"clear_terminal"`
	// Copy the last agent response to the clipboard.
	Copy *ForcedChatgptWorkspaceIDS `json:"copy"`
	// Open the shared agent-session overview.
	OpenAgents *ForcedChatgptWorkspaceIDS `json:"open_agents"`
	// Open the external editor for the current draft.
	OpenExternalEditor *ForcedChatgptWorkspaceIDS `json:"open_external_editor"`
	// Open the transcript overlay.
	OpenTranscript *ForcedChatgptWorkspaceIDS `json:"open_transcript"`
	// Queue the current composer draft while a task is running.
	Queue *ForcedChatgptWorkspaceIDS `json:"queue"`
	// Submit the current composer draft.
	Submit *ForcedChatgptWorkspaceIDS `json:"submit"`
	// Toggle Fast mode.
	ToggleFastMode *ForcedChatgptWorkspaceIDS `json:"toggle_fast_mode"`
	// Toggle raw scrollback mode for copy-friendly transcript selection.
	ToggleRawOutput *ForcedChatgptWorkspaceIDS `json:"toggle_raw_output"`
	// Toggle the composer shortcut overlay.
	ToggleShortcuts *ForcedChatgptWorkspaceIDS `json:"toggle_shortcuts"`
	// Switch between a side conversation and its parent without closing either.
	ToggleSideConversation *ForcedChatgptWorkspaceIDS `json:"toggle_side_conversation"`
	// Toggle Vim mode for the composer input.
	ToggleVimMode *ForcedChatgptWorkspaceIDS `json:"toggle_vim_mode"`
}

// List selection context keybindings for popup-style selectable lists.
type TuiListKeymap struct {
	// Accept current selection.
	Accept *ForcedChatgptWorkspaceIDS `json:"accept"`
	// Cancel and close selection view.
	Cancel *ForcedChatgptWorkspaceIDS `json:"cancel"`
	// Jump to the last list item.
	JumpBottom *ForcedChatgptWorkspaceIDS `json:"jump_bottom"`
	// Jump to the first list item.
	JumpTop *ForcedChatgptWorkspaceIDS `json:"jump_top"`
	// Move list selection down.
	MoveDown *ForcedChatgptWorkspaceIDS `json:"move_down"`
	// Move horizontally left in list pickers that support horizontal actions.
	MoveLeft *ForcedChatgptWorkspaceIDS `json:"move_left"`
	// Move horizontally right in list pickers that support horizontal actions.
	MoveRight *ForcedChatgptWorkspaceIDS `json:"move_right"`
	// Move list selection up.
	MoveUp *ForcedChatgptWorkspaceIDS `json:"move_up"`
	// Move list selection down by one page.
	PageDown *ForcedChatgptWorkspaceIDS `json:"page_down"`
	// Move list selection up by one page.
	PageUp *ForcedChatgptWorkspaceIDS `json:"page_up"`
}

// Pager context keybindings for transcript and static overlays.
type TuiPagerKeymap struct {
	// Close the pager overlay.
	Close *ForcedChatgptWorkspaceIDS `json:"close"`
	// Close the transcript overlay via its dedicated toggle key.
	CloseTranscript *ForcedChatgptWorkspaceIDS `json:"close_transcript"`
	// Scroll down by half a page.
	HalfPageDown *ForcedChatgptWorkspaceIDS `json:"half_page_down"`
	// Scroll up by half a page.
	HalfPageUp *ForcedChatgptWorkspaceIDS `json:"half_page_up"`
	// Jump to the end.
	JumpBottom *ForcedChatgptWorkspaceIDS `json:"jump_bottom"`
	// Jump to the beginning.
	JumpTop *ForcedChatgptWorkspaceIDS `json:"jump_top"`
	// Scroll down by one page.
	PageDown *ForcedChatgptWorkspaceIDS `json:"page_down"`
	// Scroll up by one page.
	PageUp *ForcedChatgptWorkspaceIDS `json:"page_up"`
	// Scroll down by one row.
	ScrollDown *ForcedChatgptWorkspaceIDS `json:"scroll_down"`
	// Scroll up by one row.
	ScrollUp *ForcedChatgptWorkspaceIDS `json:"scroll_up"`
}

// Vim normal-mode keybindings for modal editing inside text areas.
//
// Actions that use uppercase letters (like `A` for append-line-end) should be specified as
// `shift-a` in config; the runtime matcher handles cross-terminal shift-reporting
// differences automatically.
type TuiVimNormalKeymap struct {
	// Enter insert mode after cursor (`a`).
	AppendAfterCursor *ForcedChatgptWorkspaceIDS `json:"append_after_cursor"`
	// Enter insert mode at end of line (`A`).
	AppendLineEnd *ForcedChatgptWorkspaceIDS `json:"append_line_end"`
	// Cancel a pending operator and return to normal mode.
	CancelOperator *ForcedChatgptWorkspaceIDS `json:"cancel_operator"`
	// Change from cursor to end of line and enter insert mode (`C`).
	ChangeToLineEnd *ForcedChatgptWorkspaceIDS `json:"change_to_line_end"`
	// Delete character under cursor (`x`).
	DeleteChar *ForcedChatgptWorkspaceIDS `json:"delete_char"`
	// Delete from cursor to end of line (`D`).
	DeleteToLineEnd *ForcedChatgptWorkspaceIDS `json:"delete_to_line_end"`
	// Enter insert mode at cursor (`i`).
	EnterInsert *ForcedChatgptWorkspaceIDS `json:"enter_insert"`
	// Enter insert mode at first non-blank of line (`I`).
	InsertLineStart *ForcedChatgptWorkspaceIDS `json:"insert_line_start"`
	// Move cursor down (`j`), or recall newer composer history at history boundaries.
	MoveDown *ForcedChatgptWorkspaceIDS `json:"move_down"`
	// Move cursor left (`h`).
	MoveLeft *ForcedChatgptWorkspaceIDS `json:"move_left"`
	// Move cursor to end of line (`$`).
	MoveLineEnd *ForcedChatgptWorkspaceIDS `json:"move_line_end"`
	// Move cursor to start of line (`0`).
	MoveLineStart *ForcedChatgptWorkspaceIDS `json:"move_line_start"`
	// Move cursor right (`l`).
	MoveRight *ForcedChatgptWorkspaceIDS `json:"move_right"`
	// Move cursor up (`k`), or recall older composer history at history boundaries.
	MoveUp *ForcedChatgptWorkspaceIDS `json:"move_up"`
	// Move cursor to start of previous word (`b`).
	MoveWordBackward *ForcedChatgptWorkspaceIDS `json:"move_word_backward"`
	// Move cursor to end of current/next word (`e`).
	MoveWordEnd *ForcedChatgptWorkspaceIDS `json:"move_word_end"`
	// Move cursor to start of next word (`w`).
	MoveWordForward *ForcedChatgptWorkspaceIDS `json:"move_word_forward"`
	// Open a new line above and enter insert mode (`O`).
	OpenLineAbove *ForcedChatgptWorkspaceIDS `json:"open_line_above"`
	// Open a new line below and enter insert mode (`o`).
	OpenLineBelow *ForcedChatgptWorkspaceIDS `json:"open_line_below"`
	// Paste after cursor (`p`).
	PasteAfter *ForcedChatgptWorkspaceIDS `json:"paste_after"`
	// Replace the character under the cursor (`r`).
	ReplaceChar *ForcedChatgptWorkspaceIDS `json:"replace_char"`
	// Begin change operator; next keys select a text object.
	StartChangeOperator *ForcedChatgptWorkspaceIDS `json:"start_change_operator"`
	// Begin delete operator; next key selects motion (`d`).
	StartDeleteOperator *ForcedChatgptWorkspaceIDS `json:"start_delete_operator"`
	// Begin yank operator; next key selects motion (`y`).
	StartYankOperator *ForcedChatgptWorkspaceIDS `json:"start_yank_operator"`
	// Delete character under cursor and enter insert mode (`s`).
	SubstituteChar *ForcedChatgptWorkspaceIDS `json:"substitute_char"`
	// Yank the entire line (`Y`).
	YankLine *ForcedChatgptWorkspaceIDS `json:"yank_line"`
}

// Vim operator-pending keybindings for modal editing inside text areas.
//
// This context is active only while waiting for a motion after `d` or `y`. Repeating the
// operator key (`dd`, `yy`) targets the entire line. Pressing `Esc` cancels the pending
// operator and returns to normal mode without modifying text.
type TuiVimOperatorKeymap struct {
	// Cancel the pending operator and return to normal mode.
	Cancel *ForcedChatgptWorkspaceIDS `json:"cancel"`
	// Repeat delete operator to delete the whole line (`dd`).
	DeleteLine *ForcedChatgptWorkspaceIDS `json:"delete_line"`
	// Motion: down one line (`j`).
	MotionDown *ForcedChatgptWorkspaceIDS `json:"motion_down"`
	// Motion: left (`h`).
	MotionLeft *ForcedChatgptWorkspaceIDS `json:"motion_left"`
	// Motion: to end of line (`$`).
	MotionLineEnd *ForcedChatgptWorkspaceIDS `json:"motion_line_end"`
	// Motion: to start of line (`0`).
	MotionLineStart *ForcedChatgptWorkspaceIDS `json:"motion_line_start"`
	// Motion: right (`l`).
	MotionRight *ForcedChatgptWorkspaceIDS `json:"motion_right"`
	// Motion: up one line (`k`).
	MotionUp *ForcedChatgptWorkspaceIDS `json:"motion_up"`
	// Motion: to start of previous word (`b`).
	MotionWordBackward *ForcedChatgptWorkspaceIDS `json:"motion_word_backward"`
	// Motion: to end of current/next word (`e`).
	MotionWordEnd *ForcedChatgptWorkspaceIDS `json:"motion_word_end"`
	// Motion: to start of next word (`w`).
	MotionWordForward *ForcedChatgptWorkspaceIDS `json:"motion_word_forward"`
	// Select an around text object after an operator.
	SelectAroundTextObject *ForcedChatgptWorkspaceIDS `json:"select_around_text_object"`
	// Select an inner text object after an operator.
	SelectInnerTextObject *ForcedChatgptWorkspaceIDS `json:"select_inner_text_object"`
	// Repeat yank operator to yank the whole line (`yy`).
	YankLine *ForcedChatgptWorkspaceIDS `json:"yank_line"`
}

// Vim text-object keybindings for modal editing inside text areas.
type TuiVimTextObjectKeymap struct {
	// Text object: backticks.
	Backtick *ForcedChatgptWorkspaceIDS `json:"backtick"`
	// Text object: whitespace-delimited WORD.
	BigWord *ForcedChatgptWorkspaceIDS `json:"big_word"`
	// Text object: braces.
	Braces *ForcedChatgptWorkspaceIDS `json:"braces"`
	// Text object: brackets.
	Brackets *ForcedChatgptWorkspaceIDS `json:"brackets"`
	// Cancel the pending text-object command.
	Cancel *ForcedChatgptWorkspaceIDS `json:"cancel"`
	// Text object: double quotes.
	DoubleQuote *ForcedChatgptWorkspaceIDS `json:"double_quote"`
	// Text object: parentheses.
	Parentheses *ForcedChatgptWorkspaceIDS `json:"parentheses"`
	// Text object: single quotes.
	SingleQuote *ForcedChatgptWorkspaceIDS `json:"single_quote"`
	// Text object: word.
	Word *ForcedChatgptWorkspaceIDS `json:"word"`
}

// The model decides when to ask the user for approval.
//
// Never ask the user to approve commands. Failures are immediately returned to the model,
// and never escalated to the user for approval.
type AskForApprovalEnum string

const (
	AskForApprovalNever AskForApprovalEnum = "never"
	OnRequest           AskForApprovalEnum = "on-request"
)

// Configures who approval requests are routed to for review once they have been escalated.
// This does not disable separate safety checks such as ARC.
//
// Configures who approval requests are routed to for review. Examples include sandbox
// escapes, blocked network access, MCP approval prompts, and ARC escalations. Defaults to
// `user`. `auto_review` uses a carefully prompted subagent to gather relevant context and
// apply a risk-based decision framework before approving or denying the request. The legacy
// value `guardian_subagent` is accepted for compatibility.
//
// Reviewer for approval prompts unless overridden by per-app settings.
//
// Reviewer for approval prompts from this app, overriding the thread default.
type ApprovalsReviewer string

const (
	AutoReview       ApprovalsReviewer = "auto_review"
	GuardianSubagent ApprovalsReviewer = "guardian_subagent"
	User             ApprovalsReviewer = "user"
)

// Approval mode for tools unless overridden by per-app or per-tool settings.
//
// Approval mode for tools in this app unless a tool override exists.
//
// Approval mode for this tool.
//
// Approval mode for tools in this server unless a tool override exists.
type AppToolApproval string

const (
	AppToolApprovalAuto   AppToolApproval = "auto"
	AppToolApprovalPrompt AppToolApproval = "prompt"
	Approve               AppToolApproval = "approve"
	Writes                AppToolApproval = "writes"
)

type Toml string

const (
	Allow Toml = "allow"
	Deny  Toml = "deny"
)

// Preferred backend for storing CLI auth credentials. file (default): Use a file in the
// Codex home directory. keyring: Use an OS-specific keyring service. auto: Use the keyring
// if available, otherwise use a file.
//
// Determine where Codex should store CLI auth credentials.
//
// Persist credentials in CODEX_HOME/auth.json.
//
// Persist credentials in the keyring. Fail if unavailable.
//
// Use keyring when available; otherwise, fall back to a file in CODEX_HOME.
//
// Store credentials in memory only for the current process.
type AuthCredentialsStoreMode string

const (
	AuthCredentialsStoreModeAuto    AuthCredentialsStoreMode = "auto"
	AuthCredentialsStoreModeFile    AuthCredentialsStoreMode = "file"
	AuthCredentialsStoreModeKeyring AuthCredentialsStoreMode = "keyring"
	Ephemeral                       AuthCredentialsStoreMode = "ephemeral"
)

type ExperimentalThreadStoreType string

const (
	TypeLocal ExperimentalThreadStoreType = "local"
)

type CurrentTimeSource string

const (
	External CurrentTimeSource = "external"
	System   CurrentTimeSource = "system"
)

// Which inference boundaries may receive current-time reminders.
//
// Allow a reminder before any inference request once the interval is due.
//
// Allow reminders after user input or tool output; new context windows still force one.
type CurrentTimeReminderDeliveryMode string

const (
	AfterUserOrToolOutput CurrentTimeReminderDeliveryMode = "after_user_or_tool_output"
	AnyInference          CurrentTimeReminderDeliveryMode = "any_inference"
)

// Optional conversation sources available to the Guardian v2 classifier.
type GuardianV2TranscriptSource string

const (
	Reasoning   GuardianV2TranscriptSource = "reasoning"
	ToolCalls   GuardianV2TranscriptSource = "tool_calls"
	ToolOutputs GuardianV2TranscriptSource = "tool_outputs"
)

type NetworkProxyModeToml string

const (
	Full    NetworkProxyModeToml = "full"
	Limited NetworkProxyModeToml = "limited"
)

// Optional URI-based file opener. If set, citations to files in the model output will be
// hyperlinked using the specified URI scheme.
//
// Option to disable the URI-based file opener.
type URIBasedFileOpener string

const (
	Cursor                 URIBasedFileOpener = "cursor"
	URIBasedFileOpenerNone URIBasedFileOpener = "none"
	Vscode                 URIBasedFileOpener = "vscode"
	VscodeInsiders         URIBasedFileOpener = "vscode-insiders"
	Windsurf               URIBasedFileOpener = "windsurf"
)

// When set, restricts the login mechanism users may use.
type ForcedLoginMethod string

const (
	API                      ForcedLoginMethod = "api"
	ForcedLoginMethodChatgpt ForcedLoginMethod = "chatgpt"
)

// If true, history entries will not be written to disk.
//
// Save all history entries to disk.
//
// Do not write history to disk.
type HistoryPersistence string

const (
	HistoryPersistenceNone HistoryPersistence = "none"
	SaveAll                HistoryPersistence = "save-all"
)

type HookHandlerConfigType string

const (
	Agent      HookHandlerConfigType = "agent"
	Command    HookHandlerConfigType = "command"
	MCPTool    HookHandlerConfigType = "mcp_tool"
	TypePrompt HookHandlerConfigType = "prompt"
)

// Preferred backend for storing MCP OAuth credentials. keyring: Use an OS-specific keyring
// service. https://github.com/openai/codex/blob/main/codex-rs/rmcp-client/src/oauth.rs#L2
// file: Use a file in the Codex home directory. auto (default): Use the OS-specific keyring
// service if available, otherwise use a file.
//
// Determine where Codex should store and read MCP credentials.
//
// Prefer `Keyring` and use `File` when keyring storage is unavailable. Once an MCP client
// loads credentials from one store, that client keeps the resolved store for its lifetime
// so refreshes cannot switch to a possibly stale credential source. Credentials stored in
// the keyring will only be readable by Codex unless the user explicitly grants access via
// OS-level keyring access.
//
// CODEX_HOME/.credentials.json This file will be readable to Codex and other applications
// running as the same user.
//
// Keyring when available, otherwise fail.
type OAuthCredentialsStoreMode string

const (
	OAuthCredentialsStoreModeAuto    OAuthCredentialsStoreMode = "auto"
	OAuthCredentialsStoreModeFile    OAuthCredentialsStoreMode = "file"
	OAuthCredentialsStoreModeKeyring OAuthCredentialsStoreMode = "keyring"
)

// Authentication flow Codex attempts after resolving an HTTP MCP server's configured bearer
// token and authorization headers, which always take precedence. ChatGPT authentication
// falls back to stored OAuth credentials when its session provider is unavailable; both
// modes ultimately fall back to an unauthenticated connection.
//
// Use stored MCP OAuth credentials when available. Starting an OAuth login is a separate
// operation.
//
// Use the current ChatGPT session for servers on the trusted first-party ChatGPT origin. If
// no ChatGPT session provider is available, startup can still fall back to stored OAuth
// credentials.
type MCPServerAuth string

const (
	MCPServerAuthChatgpt MCPServerAuth = "chatgpt"
	Oauth                MCPServerAuth = "oauth"
)

// A model-facing surface on which a tool can be exposed.
//
// Nested tools available to Code Mode scripts.
//
// Tools discovered later through tool search.
//
// Tools present in the model's initial tool list.
type ToolExposureSurface string

const (
	CodeMode ToolExposureSurface = "code_mode"
	Deferred ToolExposureSurface = "deferred"
	Direct   ToolExposureSurface = "direct"
)

// Source kind used to install this marketplace.
type MarketplaceSourceType string

const (
	Git                        MarketplaceSourceType = "git"
	MarketplaceSourceTypeLocal MarketplaceSourceType = "local"
)

// Controls whether the auto-compaction limit applies to the full context or only to tokens
// after the carried prefix in the current compaction window.
//
// Selects which part of the active context is charged against
// `model_auto_compact_token_limit`.
//
// Count the full active context against the limit.
//
// Count sampled output and later growth after the carried window prefix.
type AutoCompactTokenLimitScope string

const (
	BodyAfterPrefix AutoCompactTokenLimitScope = "body_after_prefix"
	Total           AutoCompactTokenLimitScope = "total"
)

// Which wire protocol this provider expects.
//
// Wire protocol that the provider speaks.
//
// The Responses API exposed by OpenAI at `/v1/responses`.
type WireAPI string

const (
	Responses WireAPI = "responses"
)

// A summary of the reasoning performed by the model. This can be useful for debugging and
// understanding the model's reasoning process. See
// https://platform.openai.com/docs/guides/reasoning?api-mode=responses#reasoning-summaries
//
// Option to disable reasoning summaries.
type ReasoningSummary string

const (
	Concise              ReasoningSummary = "concise"
	Detailed             ReasoningSummary = "detailed"
	ReasoningSummaryAuto ReasoningSummary = "auto"
	ReasoningSummaryNone ReasoningSummary = "none"
)

// Optional verbosity control for GPT-5 models (Responses API `text.verbosity`).
//
// Controls output length/detail on GPT-5 models via the Responses API. Serialized with
// lowercase values to match the OpenAI API.
type Verbosity string

const (
	High   Verbosity = "high"
	Low    Verbosity = "low"
	Medium Verbosity = "medium"
)

// Binary payload
//
// JSON payload
type OtelHTTPProtocol string

const (
	Binary OtelHTTPProtocol = "binary"
	JSON   OtelHTTPProtocol = "json"
)

type OtelExporterKindEnum string

const (
	OtelExporterKindNone OtelExporterKindEnum = "none"
	Statsig              OtelExporterKindEnum = "statsig"
)

// Optionally specify a personality for the model
type Personality string

const (
	Friendly        Personality = "friendly"
	PersonalityNone Personality = "none"
	Pragmatic       Personality = "pragmatic"
)

// Sandbox mode to use.
type SandboxMode string

const (
	DangerFullAccess SandboxMode = "danger-full-access"
	ReadOnly         SandboxMode = "read-only"
	WorkspaceWrite   SandboxMode = "workspace-write"
)

// Preferred layout for resume/fork session picker results.
//
// Preferred layout for the resume/fork session picker.
type SessionPickerViewMode string

const (
	Comfortable SessionPickerViewMode = "comfortable"
	Dense       SessionPickerViewMode = "dense"
)

// Controls the web search tool mode: disabled, cached, indexed, or live.
type WebSearchMode string

const (
	Cached   WebSearchMode = "cached"
	Disabled WebSearchMode = "disabled"
	Indexed  WebSearchMode = "indexed"
	Live     WebSearchMode = "live"
)

type WindowsSandboxModeToml string

const (
	Elevated   WindowsSandboxModeToml = "elevated"
	Unelevated WindowsSandboxModeToml = "unelevated"
)

// Represents the trust level for a project directory. This determines the approval policy
// and sandbox mode applied.
type TrustLevel string

const (
	Trusted   TrustLevel = "trusted"
	Untrusted TrustLevel = "untrusted"
)

type RealtimeTransport string

const (
	Webrtc    RealtimeTransport = "webrtc"
	Websocket RealtimeTransport = "websocket"
)

type RealtimeWsMode string

const (
	Conversational RealtimeWsMode = "conversational"
	Transcription  RealtimeWsMode = "transcription"
)

type RealtimeConversationVersion string

const (
	V1 RealtimeConversationVersion = "v1"
	V2 RealtimeConversationVersion = "v2"
	V3 RealtimeConversationVersion = "v3"
)

type RealtimeVoice string

const (
	Alloy   RealtimeVoice = "alloy"
	Arbor   RealtimeVoice = "arbor"
	Ash     RealtimeVoice = "ash"
	Ballad  RealtimeVoice = "ballad"
	Breeze  RealtimeVoice = "breeze"
	Cedar   RealtimeVoice = "cedar"
	Coral   RealtimeVoice = "coral"
	Cove    RealtimeVoice = "cove"
	Echo    RealtimeVoice = "echo"
	Ember   RealtimeVoice = "ember"
	Juniper RealtimeVoice = "juniper"
	Maple   RealtimeVoice = "maple"
	Marin   RealtimeVoice = "marin"
	Sage    RealtimeVoice = "sage"
	Shimmer RealtimeVoice = "shimmer"
	Sol     RealtimeVoice = "sol"
	Spruce  RealtimeVoice = "spruce"
	Vale    RealtimeVoice = "vale"
	Verse   RealtimeVoice = "verse"
)

// Assigns a shell environment variable pattern to the include-only or exclude set. Includes
// do not re-add variables removed by another exclude pattern.
type ShellEnvironmentPolicyFilter string

const (
	Exclude ShellEnvironmentPolicyFilter = "exclude"
	Include ShellEnvironmentPolicyFilter = "include"
)

// "Core" environment variables for the platform. On UNIX, this would include HOME, LOGNAME,
// PATH, SHELL, and USER, among others.
//
// Inherits the full environment from the parent process.
//
// Do not inherit any environment variables from the parent process.
type ShellEnvironmentPolicyInherit string

const (
	All                               ShellEnvironmentPolicyInherit = "all"
	Core                              ShellEnvironmentPolicyInherit = "core"
	ShellEnvironmentPolicyInheritNone ShellEnvironmentPolicyInherit = "none"
)

type ToolSuggestDiscoverableType string

const (
	Connector ToolSuggestDiscoverableType = "connector"
	Plugin    ToolSuggestDiscoverableType = "plugin"
)

// Controls whether the TUI uses the terminal's alternate screen buffer.
//
// - `auto` (default): Use alternate screen. - `always`: Always use alternate screen. -
// `never`: Never use alternate screen (inline mode only, preserves scrollback).
//
// Controls whether the TUI uses the terminal's alternate screen buffer.
//
// - `auto` (default): Use alternate screen mode. - `always`: Always use alternate screen
// mode. - `never`: Never use alternate screen mode. Runs in inline mode, preserving
// scrollback.
//
// The CLI flag `--no-alt-screen` can override this setting at runtime.
//
// Use alternate screen mode.
//
// Always use alternate screen mode.
//
// Never use alternate screen (inline mode only).
type AltScreenMode string

const (
	AltScreenModeAlways AltScreenMode = "always"
	AltScreenModeAuto   AltScreenMode = "auto"
	AltScreenModeNever  AltScreenMode = "never"
)

// Controls whether TUI notifications are delivered only when the terminal is unfocused or
// regardless of focus. Defaults to `unfocused`.
//
// Emit TUI notifications only while the terminal is unfocused.
//
// Emit TUI notifications regardless of terminal focus.
type NotificationCondition string

const (
	NotificationConditionAlways NotificationCondition = "always"
	Unfocused                   NotificationCondition = "unfocused"
)

// Notification method to use for terminal notifications. Defaults to `auto`.
type NotificationMethod string

const (
	Bel                    NotificationMethod = "bel"
	NotificationMethodAuto NotificationMethod = "auto"
	Osc9                   NotificationMethod = "osc9"
)

// Where the terminal pet should anchor vertically.
//
// Defaults to `composer`, which follows the current TUI composer viewport.
//
// Anchor the pet to the bottom of the current TUI composer viewport.
//
// Anchor the pet to the physical bottom of the terminal screen.
type TuiPetAnchor string

const (
	Composer     TuiPetAnchor = "composer"
	ScreenBottom TuiPetAnchor = "screen-bottom"
)

// Working directory to use when resuming or forking a session. When unset, prompt if the
// current and session directories differ.
//
// Working directory to use when resuming or forking a session.
//
// Use the directory where Codex was launched.
//
// Use the latest working directory recorded in the selected session.
type ResumeCwdMode string

const (
	Current ResumeCwdMode = "current"
	Session ResumeCwdMode = "session"
)

type AgentValue struct {
	AgentRoleToml *AgentRoleToml
	Bool          *bool
	Integer       *int64
	String        *string
}

func (x *AgentValue) UnmarshalJSON(data []byte) error {
	x.AgentRoleToml = nil
	var c AgentRoleToml
	object, err := unmarshalUnion(data, &x.Integer, nil, &x.Bool, &x.String, false, nil, true, &c, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
		x.AgentRoleToml = &c
	}
	return nil
}

func (x *AgentValue) MarshalJSON() ([]byte, error) {
	return marshalUnion(x.Integer, nil, x.Bool, x.String, false, nil, x.AgentRoleToml != nil, x.AgentRoleToml, false, nil, false, nil, false)
}

// Default approval policy for executing commands.
//
// Determines the conditions under which the user is consulted to approve running the
// command proposed by Codex.
type AskForApproval struct {
	AskForApprovalClass *AskForApprovalClass
	Enum                *AskForApprovalEnum
}

func (x *AskForApproval) UnmarshalJSON(data []byte) error {
	x.AskForApprovalClass = nil
	x.Enum = nil
	var c AskForApprovalClass
	object, err := unmarshalUnion(data, nil, nil, nil, nil, false, nil, true, &c, false, nil, true, &x.Enum, false)
	if err != nil {
		return err
	}
	if object {
		x.AskForApprovalClass = &c
	}
	return nil
}

func (x *AskForApproval) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, nil, false, nil, x.AskForApprovalClass != nil, x.AskForApprovalClass, false, nil, x.Enum != nil, x.Enum, false)
}

type TentacledAppsMCPPathOverride struct {
	Bool                      *bool
	PurpleAppsMCPPathOverride *PurpleAppsMCPPathOverride
}

func (x *TentacledAppsMCPPathOverride) UnmarshalJSON(data []byte) error {
	x.PurpleAppsMCPPathOverride = nil
	var c PurpleAppsMCPPathOverride
	object, err := unmarshalUnion(data, nil, nil, &x.Bool, nil, false, nil, true, &c, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
		x.PurpleAppsMCPPathOverride = &c
	}
	return nil
}

func (x *TentacledAppsMCPPathOverride) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, x.Bool, nil, false, nil, x.PurpleAppsMCPPathOverride != nil, x.PurpleAppsMCPPathOverride, false, nil, false, nil, false)
}

type FeatureTomlForCodeModeConfigToml struct {
	Bool               *bool
	CodeModeConfigToml *CodeModeConfigToml
}

func (x *FeatureTomlForCodeModeConfigToml) UnmarshalJSON(data []byte) error {
	x.CodeModeConfigToml = nil
	var c CodeModeConfigToml
	object, err := unmarshalUnion(data, nil, nil, &x.Bool, nil, false, nil, true, &c, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
		x.CodeModeConfigToml = &c
	}
	return nil
}

func (x *FeatureTomlForCodeModeConfigToml) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, x.Bool, nil, false, nil, x.CodeModeConfigToml != nil, x.CodeModeConfigToml, false, nil, false, nil, false)
}

type FeatureTomlForCodeModeHostConfigToml struct {
	Bool                   *bool
	CodeModeHostConfigToml *CodeModeHostConfigToml
}

func (x *FeatureTomlForCodeModeHostConfigToml) UnmarshalJSON(data []byte) error {
	x.CodeModeHostConfigToml = nil
	var c CodeModeHostConfigToml
	object, err := unmarshalUnion(data, nil, nil, &x.Bool, nil, false, nil, true, &c, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
		x.CodeModeHostConfigToml = &c
	}
	return nil
}

func (x *FeatureTomlForCodeModeHostConfigToml) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, x.Bool, nil, false, nil, x.CodeModeHostConfigToml != nil, x.CodeModeHostConfigToml, false, nil, false, nil, false)
}

type FeatureTomlForCurrentTimeReminderConfigToml struct {
	Bool                          *bool
	CurrentTimeReminderConfigToml *CurrentTimeReminderConfigToml
}

func (x *FeatureTomlForCurrentTimeReminderConfigToml) UnmarshalJSON(data []byte) error {
	x.CurrentTimeReminderConfigToml = nil
	var c CurrentTimeReminderConfigToml
	object, err := unmarshalUnion(data, nil, nil, &x.Bool, nil, false, nil, true, &c, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
		x.CurrentTimeReminderConfigToml = &c
	}
	return nil
}

func (x *FeatureTomlForCurrentTimeReminderConfigToml) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, x.Bool, nil, false, nil, x.CurrentTimeReminderConfigToml != nil, x.CurrentTimeReminderConfigToml, false, nil, false, nil, false)
}

type FeatureTomlForGuardianV2ConfigToml struct {
	Bool                 *bool
	GuardianV2ConfigToml *GuardianV2ConfigToml
}

func (x *FeatureTomlForGuardianV2ConfigToml) UnmarshalJSON(data []byte) error {
	x.GuardianV2ConfigToml = nil
	var c GuardianV2ConfigToml
	object, err := unmarshalUnion(data, nil, nil, &x.Bool, nil, false, nil, true, &c, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
		x.GuardianV2ConfigToml = &c
	}
	return nil
}

func (x *FeatureTomlForGuardianV2ConfigToml) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, x.Bool, nil, false, nil, x.GuardianV2ConfigToml != nil, x.GuardianV2ConfigToml, false, nil, false, nil, false)
}

type FeatureTomlForMultiAgentV2ConfigToml struct {
	Bool                   *bool
	MultiAgentV2ConfigToml *MultiAgentV2ConfigToml
}

func (x *FeatureTomlForMultiAgentV2ConfigToml) UnmarshalJSON(data []byte) error {
	x.MultiAgentV2ConfigToml = nil
	var c MultiAgentV2ConfigToml
	object, err := unmarshalUnion(data, nil, nil, &x.Bool, nil, false, nil, true, &c, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
		x.MultiAgentV2ConfigToml = &c
	}
	return nil
}

func (x *FeatureTomlForMultiAgentV2ConfigToml) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, x.Bool, nil, false, nil, x.MultiAgentV2ConfigToml != nil, x.MultiAgentV2ConfigToml, false, nil, false, nil, false)
}

type FeatureTomlForNetworkProxyConfigToml struct {
	Bool                   *bool
	NetworkProxyConfigToml *NetworkProxyConfigToml
}

func (x *FeatureTomlForNetworkProxyConfigToml) UnmarshalJSON(data []byte) error {
	x.NetworkProxyConfigToml = nil
	var c NetworkProxyConfigToml
	object, err := unmarshalUnion(data, nil, nil, &x.Bool, nil, false, nil, true, &c, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
		x.NetworkProxyConfigToml = &c
	}
	return nil
}

func (x *FeatureTomlForNetworkProxyConfigToml) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, x.Bool, nil, false, nil, x.NetworkProxyConfigToml != nil, x.NetworkProxyConfigToml, false, nil, false, nil, false)
}

type FeatureTomlForNonPrefixedMCPToolNamesConfigToml struct {
	Bool                              *bool
	NonPrefixedMCPToolNamesConfigToml *NonPrefixedMCPToolNamesConfigToml
}

func (x *FeatureTomlForNonPrefixedMCPToolNamesConfigToml) UnmarshalJSON(data []byte) error {
	x.NonPrefixedMCPToolNamesConfigToml = nil
	var c NonPrefixedMCPToolNamesConfigToml
	object, err := unmarshalUnion(data, nil, nil, &x.Bool, nil, false, nil, true, &c, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
		x.NonPrefixedMCPToolNamesConfigToml = &c
	}
	return nil
}

func (x *FeatureTomlForNonPrefixedMCPToolNamesConfigToml) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, x.Bool, nil, false, nil, x.NonPrefixedMCPToolNamesConfigToml != nil, x.NonPrefixedMCPToolNamesConfigToml, false, nil, false, nil, false)
}

type FeatureTomlForRolloutBudgetConfigToml struct {
	Bool                    *bool
	RolloutBudgetConfigToml *RolloutBudgetConfigToml
}

func (x *FeatureTomlForRolloutBudgetConfigToml) UnmarshalJSON(data []byte) error {
	x.RolloutBudgetConfigToml = nil
	var c RolloutBudgetConfigToml
	object, err := unmarshalUnion(data, nil, nil, &x.Bool, nil, false, nil, true, &c, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
		x.RolloutBudgetConfigToml = &c
	}
	return nil
}

func (x *FeatureTomlForRolloutBudgetConfigToml) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, x.Bool, nil, false, nil, x.RolloutBudgetConfigToml != nil, x.RolloutBudgetConfigToml, false, nil, false, nil, false)
}

type FeatureTomlForTokenBudgetConfigToml struct {
	Bool                  *bool
	TokenBudgetConfigToml *TokenBudgetConfigToml
}

func (x *FeatureTomlForTokenBudgetConfigToml) UnmarshalJSON(data []byte) error {
	x.TokenBudgetConfigToml = nil
	var c TokenBudgetConfigToml
	object, err := unmarshalUnion(data, nil, nil, &x.Bool, nil, false, nil, true, &c, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
		x.TokenBudgetConfigToml = &c
	}
	return nil
}

func (x *FeatureTomlForTokenBudgetConfigToml) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, x.Bool, nil, false, nil, x.TokenBudgetConfigToml != nil, x.TokenBudgetConfigToml, false, nil, false, nil, false)
}

// When set, restricts ChatGPT login to one or more workspace identifiers.
//
// Backward-compatible shape for ChatGPT workspace login restrictions in config.toml.
//
// Start composing a new agent task.
//
// One action binding value in config.
//
// This accepts either:
//
// 1. A single key or chord string (`"ctrl-a"` or `"ctrl-x ctrl-s"`). 2. A list of
// alternative bindings (`["ctrl-a", "ctrl-x ctrl-s"]`).
//
// An empty list explicitly unbinds the action in that scope. Because an explicit empty list
// is still a configured value, runtime resolution must not fall through to global or
// built-in defaults for that action.
//
// Rename the selected task.
//
// Search the available agent tasks.
//
// Stop the selected running task.
//
// Toggle grouping tasks by status or project.
//
// Approve the primary option.
//
// Approve with exec-policy prefix when that option exists.
//
// Approve for session when that option exists.
//
// Cancel an elicitation request.
//
// Decline and provide corrective guidance.
//
// Deny without providing follow-up guidance.
//
// Open the full-screen approval details view.
//
// Open the thread that requested approval when shown from another thread.
//
// Decrease the active reasoning effort.
//
// Edit the most recently queued message.
//
// Increase the active reasoning effort.
//
// Interrupt the active turn.
//
// Switch to the next available permission mode.
//
// Switch to the previous available permission mode.
//
// Toggle the microphone in an active voice conversation.
//
// Move to the next match in reverse history search.
//
// Open reverse history search or move to the previous match.
//
// Queue the current composer draft while a task is running.
//
// Submit the current composer draft.
//
// Toggle the composer shortcut overlay.
//
// Delete one grapheme to the left.
//
// Delete the previous word.
//
// Delete one grapheme to the right.
//
// Delete the next word.
//
// Insert a newline in the editor.
//
// Kill text from cursor to line end.
//
// Kill text from cursor to line start.
//
// Kill the current line.
//
// Move cursor down one visual line.
//
// Move cursor left by one grapheme.
//
// Move cursor to end of line.
//
// Move cursor to beginning of line.
//
// Move cursor right by one grapheme.
//
// Move cursor up one visual line.
//
// Move cursor to beginning of previous word.
//
// Move cursor to end of next word.
//
// Yank the kill buffer.
//
// Clear the terminal UI.
//
// Copy the last agent response to the clipboard.
//
// Open the shared agent-session overview.
//
// Open the external editor for the current draft.
//
// Open the transcript overlay.
//
// Toggle Fast mode.
//
// Toggle raw scrollback mode for copy-friendly transcript selection.
//
// Switch between a side conversation and its parent without closing either.
//
// Toggle Vim mode for the composer input.
//
// Accept current selection.
//
// Cancel and close selection view.
//
// Jump to the last list item.
//
// Jump to the first list item.
//
// Move list selection down.
//
// Move horizontally left in list pickers that support horizontal actions.
//
// Move horizontally right in list pickers that support horizontal actions.
//
// Move list selection up.
//
// Move list selection down by one page.
//
// Move list selection up by one page.
//
// Close the pager overlay.
//
// Close the transcript overlay via its dedicated toggle key.
//
// Scroll down by half a page.
//
// Scroll up by half a page.
//
// Jump to the end.
//
// Jump to the beginning.
//
// Scroll down by one page.
//
// Scroll up by one page.
//
// Scroll down by one row.
//
// Scroll up by one row.
//
// Enter insert mode after cursor (`a`).
//
// Enter insert mode at end of line (`A`).
//
// Cancel a pending operator and return to normal mode.
//
// Change from cursor to end of line and enter insert mode (`C`).
//
// Delete character under cursor (`x`).
//
// Delete from cursor to end of line (`D`).
//
// Enter insert mode at cursor (`i`).
//
// Enter insert mode at first non-blank of line (`I`).
//
// Move cursor down (`j`), or recall newer composer history at history boundaries.
//
// Move cursor left (`h`).
//
// Move cursor to end of line (`$`).
//
// Move cursor to start of line (`0`).
//
// Move cursor right (`l`).
//
// Move cursor up (`k`), or recall older composer history at history boundaries.
//
// Move cursor to start of previous word (`b`).
//
// Move cursor to end of current/next word (`e`).
//
// Move cursor to start of next word (`w`).
//
// Open a new line above and enter insert mode (`O`).
//
// Open a new line below and enter insert mode (`o`).
//
// Paste after cursor (`p`).
//
// Replace the character under the cursor (`r`).
//
// Begin change operator; next keys select a text object.
//
// Begin delete operator; next key selects motion (`d`).
//
// Begin yank operator; next key selects motion (`y`).
//
// Delete character under cursor and enter insert mode (`s`).
//
// Yank the entire line (`Y`).
//
// Cancel the pending operator and return to normal mode.
//
// Repeat delete operator to delete the whole line (`dd`).
//
// Motion: down one line (`j`).
//
// Motion: left (`h`).
//
// Motion: to end of line (`$`).
//
// Motion: to start of line (`0`).
//
// Motion: right (`l`).
//
// Motion: up one line (`k`).
//
// Motion: to start of previous word (`b`).
//
// Motion: to end of current/next word (`e`).
//
// Motion: to start of next word (`w`).
//
// Select an around text object after an operator.
//
// Select an inner text object after an operator.
//
// Repeat yank operator to yank the whole line (`yy`).
//
// Text object: backticks.
//
// Text object: whitespace-delimited WORD.
//
// Text object: braces.
//
// Text object: brackets.
//
// Cancel the pending text-object command.
//
// Text object: double quotes.
//
// Text object: parentheses.
//
// Text object: single quotes.
//
// Text object: word.
type ForcedChatgptWorkspaceIDS struct {
	String      *string
	StringArray []string
}

func (x *ForcedChatgptWorkspaceIDS) UnmarshalJSON(data []byte) error {
	x.StringArray = nil
	object, err := unmarshalUnion(data, nil, nil, nil, &x.String, true, &x.StringArray, false, nil, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
	}
	return nil
}

func (x *ForcedChatgptWorkspaceIDS) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, x.String, x.StringArray != nil, x.StringArray, false, nil, false, nil, false, nil, false)
}

type MCPServerEnvVar struct {
	MCPServerEnvVarClass *MCPServerEnvVarClass
	String               *string
}

func (x *MCPServerEnvVar) UnmarshalJSON(data []byte) error {
	x.MCPServerEnvVarClass = nil
	var c MCPServerEnvVarClass
	object, err := unmarshalUnion(data, nil, nil, nil, &x.String, false, nil, true, &c, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
		x.MCPServerEnvVarClass = &c
	}
	return nil
}

func (x *MCPServerEnvVar) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, x.String, false, nil, x.MCPServerEnvVarClass != nil, x.MCPServerEnvVarClass, false, nil, false, nil, false)
}

// Optional log exporter
//
// Which OTEL exporter to use.
//
// # Optional metrics exporter
//
// Optional trace exporter
type OtelExporterKind struct {
	Enum                  *OtelExporterKindEnum
	OtelExporterKindClass *OtelExporterKindClass
}

func (x *OtelExporterKind) UnmarshalJSON(data []byte) error {
	x.OtelExporterKindClass = nil
	x.Enum = nil
	var c OtelExporterKindClass
	object, err := unmarshalUnion(data, nil, nil, nil, nil, false, nil, true, &c, false, nil, true, &x.Enum, false)
	if err != nil {
		return err
	}
	if object {
		x.OtelExporterKindClass = &c
	}
	return nil
}

func (x *OtelExporterKind) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, nil, false, nil, x.OtelExporterKindClass != nil, x.OtelExporterKindClass, false, nil, x.Enum != nil, x.Enum, false)
}

type StickyAppsMCPPathOverride struct {
	Bool                      *bool
	FluffyAppsMCPPathOverride *FluffyAppsMCPPathOverride
}

func (x *StickyAppsMCPPathOverride) UnmarshalJSON(data []byte) error {
	x.FluffyAppsMCPPathOverride = nil
	var c FluffyAppsMCPPathOverride
	object, err := unmarshalUnion(data, nil, nil, &x.Bool, nil, false, nil, true, &c, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
		x.FluffyAppsMCPPathOverride = &c
	}
	return nil
}

func (x *StickyAppsMCPPathOverride) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, x.Bool, nil, false, nil, x.FluffyAppsMCPPathOverride != nil, x.FluffyAppsMCPPathOverride, false, nil, false, nil, false)
}

// Enable desktop notifications from the TUI. Defaults to `true`.
type Notifications struct {
	Bool        *bool
	StringArray []string
}

func (x *Notifications) UnmarshalJSON(data []byte) error {
	x.StringArray = nil
	object, err := unmarshalUnion(data, nil, nil, &x.Bool, nil, true, &x.StringArray, false, nil, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
	}
	return nil
}

func (x *Notifications) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, x.Bool, nil, x.StringArray != nil, x.StringArray, false, nil, false, nil, false, nil, false)
}

func unmarshalUnion(data []byte, pi **int64, pf **float64, pb **bool, ps **string, haveArray bool, pa interface{}, haveObject bool, pc interface{}, haveMap bool, pm interface{}, haveEnum bool, pe interface{}, nullable bool) (bool, error) {
	if pi != nil {
		*pi = nil
	}
	if pf != nil {
		*pf = nil
	}
	if pb != nil {
		*pb = nil
	}
	if ps != nil {
		*ps = nil
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return false, err
	}

	switch v := tok.(type) {
	case json.Number:
		if pi != nil {
			i, err := v.Int64()
			if err == nil {
				*pi = &i
				return false, nil
			}
		}
		if pf != nil {
			f, err := v.Float64()
			if err == nil {
				*pf = &f
				return false, nil
			}
			return false, errors.New("Unparsable number")
		}
		return false, errors.New("Union does not contain number")
	case float64:
		return false, errors.New("Decoder should not return float64")
	case bool:
		if pb != nil {
			*pb = &v
			return false, nil
		}
		return false, errors.New("Union does not contain bool")
	case string:
		if haveEnum {
			return false, json.Unmarshal(data, pe)
		}
		if ps != nil {
			*ps = &v
			return false, nil
		}
		return false, errors.New("Union does not contain string")
	case nil:
		if nullable {
			return false, nil
		}
		return false, errors.New("Union does not contain null")
	case json.Delim:
		if v == '{' {
			if haveObject {
				return true, json.Unmarshal(data, pc)
			}
			if haveMap {
				return false, json.Unmarshal(data, pm)
			}
			return false, errors.New("Union does not contain object")
		}
		if v == '[' {
			if haveArray {
				return false, json.Unmarshal(data, pa)
			}
			return false, errors.New("Union does not contain array")
		}
		return false, errors.New("Cannot handle delimiter")
	}
	return false, errors.New("Cannot unmarshal union")
}

func marshalUnion(pi *int64, pf *float64, pb *bool, ps *string, haveArray bool, pa interface{}, haveObject bool, pc interface{}, haveMap bool, pm interface{}, haveEnum bool, pe interface{}, nullable bool) ([]byte, error) {
	if pi != nil {
		return json.Marshal(*pi)
	}
	if pf != nil {
		return json.Marshal(*pf)
	}
	if pb != nil {
		return json.Marshal(*pb)
	}
	if ps != nil {
		return json.Marshal(*ps)
	}
	if haveArray {
		return json.Marshal(pa)
	}
	if haveObject {
		return json.Marshal(pc)
	}
	if haveMap {
		return json.Marshal(pm)
	}
	if haveEnum {
		return json.Marshal(pe)
	}
	if nullable {
		return json.Marshal(nil)
	}
	return nil, errors.New("Union must not be null")
}
