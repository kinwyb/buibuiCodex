// Code generated from JSON Schema using quicktype. DO NOT EDIT.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    applyPatchApprovalParams, err := UnmarshalApplyPatchApprovalParams(bytes)
//    bytes, err = applyPatchApprovalParams.Marshal()
//
//    applyPatchApprovalResponse, err := UnmarshalApplyPatchApprovalResponse(bytes)
//    bytes, err = applyPatchApprovalResponse.Marshal()
//
//    attestationGenerateParams, err := UnmarshalAttestationGenerateParams(bytes)
//    bytes, err = attestationGenerateParams.Marshal()
//
//    attestationGenerateResponse, err := UnmarshalAttestationGenerateResponse(bytes)
//    bytes, err = attestationGenerateResponse.Marshal()
//
//    chatgptAuthTokensRefreshParams, err := UnmarshalChatgptAuthTokensRefreshParams(bytes)
//    bytes, err = chatgptAuthTokensRefreshParams.Marshal()
//
//    chatgptAuthTokensRefreshResponse, err := UnmarshalChatgptAuthTokensRefreshResponse(bytes)
//    bytes, err = chatgptAuthTokensRefreshResponse.Marshal()
//
//    clientNotification, err := UnmarshalClientNotification(bytes)
//    bytes, err = clientNotification.Marshal()
//
//    clientRequest, err := UnmarshalClientRequest(bytes)
//    bytes, err = clientRequest.Marshal()
//
//    codexAppServerProtocol, err := UnmarshalCodexAppServerProtocol(bytes)
//    bytes, err = codexAppServerProtocol.Marshal()
//
//    codexAppServerProtocolV2, err := UnmarshalCodexAppServerProtocolV2(bytes)
//    bytes, err = codexAppServerProtocolV2.Marshal()
//
//    commandExecutionRequestApprovalParams, err := UnmarshalCommandExecutionRequestApprovalParams(bytes)
//    bytes, err = commandExecutionRequestApprovalParams.Marshal()
//
//    commandExecutionRequestApprovalResponse, err := UnmarshalCommandExecutionRequestApprovalResponse(bytes)
//    bytes, err = commandExecutionRequestApprovalResponse.Marshal()
//
//    dynamicToolCallParams, err := UnmarshalDynamicToolCallParams(bytes)
//    bytes, err = dynamicToolCallParams.Marshal()
//
//    dynamicToolCallResponse, err := UnmarshalDynamicToolCallResponse(bytes)
//    bytes, err = dynamicToolCallResponse.Marshal()
//
//    execCommandApprovalParams, err := UnmarshalExecCommandApprovalParams(bytes)
//    bytes, err = execCommandApprovalParams.Marshal()
//
//    execCommandApprovalResponse, err := UnmarshalExecCommandApprovalResponse(bytes)
//    bytes, err = execCommandApprovalResponse.Marshal()
//
//    fileChangeRequestApprovalParams, err := UnmarshalFileChangeRequestApprovalParams(bytes)
//    bytes, err = fileChangeRequestApprovalParams.Marshal()
//
//    fileChangeRequestApprovalResponse, err := UnmarshalFileChangeRequestApprovalResponse(bytes)
//    bytes, err = fileChangeRequestApprovalResponse.Marshal()
//
//    fuzzyFileSearchParams, err := UnmarshalFuzzyFileSearchParams(bytes)
//    bytes, err = fuzzyFileSearchParams.Marshal()
//
//    fuzzyFileSearchResponse, err := UnmarshalFuzzyFileSearchResponse(bytes)
//    bytes, err = fuzzyFileSearchResponse.Marshal()
//
//    fuzzyFileSearchSessionCompletedNotification, err := UnmarshalFuzzyFileSearchSessionCompletedNotification(bytes)
//    bytes, err = fuzzyFileSearchSessionCompletedNotification.Marshal()
//
//    fuzzyFileSearchSessionUpdatedNotification, err := UnmarshalFuzzyFileSearchSessionUpdatedNotification(bytes)
//    bytes, err = fuzzyFileSearchSessionUpdatedNotification.Marshal()
//
//    jSONRPCError, err := UnmarshalJSONRPCError(bytes)
//    bytes, err = jSONRPCError.Marshal()
//
//    jSONRPCErrorError, err := UnmarshalJSONRPCErrorError(bytes)
//    bytes, err = jSONRPCErrorError.Marshal()
//
//    jSONRPCMessage, err := UnmarshalJSONRPCMessage(bytes)
//    bytes, err = jSONRPCMessage.Marshal()
//
//    jSONRPCNotification, err := UnmarshalJSONRPCNotification(bytes)
//    bytes, err = jSONRPCNotification.Marshal()
//
//    jSONRPCRequest, err := UnmarshalJSONRPCRequest(bytes)
//    bytes, err = jSONRPCRequest.Marshal()
//
//    jSONRPCResponse, err := UnmarshalJSONRPCResponse(bytes)
//    bytes, err = jSONRPCResponse.Marshal()
//
//    mCPServerElicitationRequestParams, err := UnmarshalMCPServerElicitationRequestParams(bytes)
//    bytes, err = mCPServerElicitationRequestParams.Marshal()
//
//    mCPServerElicitationRequestResponse, err := UnmarshalMCPServerElicitationRequestResponse(bytes)
//    bytes, err = mCPServerElicitationRequestResponse.Marshal()
//
//    permissionsRequestApprovalParams, err := UnmarshalPermissionsRequestApprovalParams(bytes)
//    bytes, err = permissionsRequestApprovalParams.Marshal()
//
//    permissionsRequestApprovalResponse, err := UnmarshalPermissionsRequestApprovalResponse(bytes)
//    bytes, err = permissionsRequestApprovalResponse.Marshal()
//
//    requestID, err := UnmarshalRequestID(bytes)
//    bytes, err = requestID.Marshal()
//
//    serverNotification, err := UnmarshalServerNotification(bytes)
//    bytes, err = serverNotification.Marshal()
//
//    serverRequest, err := UnmarshalServerRequest(bytes)
//    bytes, err = serverRequest.Marshal()
//
//    toolRequestUserInputParams, err := UnmarshalToolRequestUserInputParams(bytes)
//    bytes, err = toolRequestUserInputParams.Marshal()
//
//    toolRequestUserInputResponse, err := UnmarshalToolRequestUserInputResponse(bytes)
//    bytes, err = toolRequestUserInputResponse.Marshal()

package schema

import "bytes"
import "errors"

import "encoding/json"

func UnmarshalApplyPatchApprovalParams(data []byte) (ApplyPatchApprovalParams, error) {
	var r ApplyPatchApprovalParams
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ApplyPatchApprovalParams) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalApplyPatchApprovalResponse(data []byte) (ApplyPatchApprovalResponse, error) {
	var r ApplyPatchApprovalResponse
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ApplyPatchApprovalResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type AttestationGenerateParams map[string]interface{}

func UnmarshalAttestationGenerateParams(data []byte) (AttestationGenerateParams, error) {
	var r AttestationGenerateParams
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *AttestationGenerateParams) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalAttestationGenerateResponse(data []byte) (AttestationGenerateResponse, error) {
	var r AttestationGenerateResponse
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *AttestationGenerateResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalChatgptAuthTokensRefreshParams(data []byte) (ChatgptAuthTokensRefreshParams, error) {
	var r ChatgptAuthTokensRefreshParams
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ChatgptAuthTokensRefreshParams) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalChatgptAuthTokensRefreshResponse(data []byte) (ChatgptAuthTokensRefreshResponse, error) {
	var r ChatgptAuthTokensRefreshResponse
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ChatgptAuthTokensRefreshResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalClientNotification(data []byte) (ClientNotification, error) {
	var r ClientNotification
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ClientNotification) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalClientRequest(data []byte) (ClientRequest, error) {
	var r ClientRequest
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ClientRequest) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type CodexAppServerProtocol map[string]interface{}

func UnmarshalCodexAppServerProtocol(data []byte) (CodexAppServerProtocol, error) {
	var r CodexAppServerProtocol
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *CodexAppServerProtocol) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type CodexAppServerProtocolV2 map[string]interface{}

func UnmarshalCodexAppServerProtocolV2(data []byte) (CodexAppServerProtocolV2, error) {
	var r CodexAppServerProtocolV2
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *CodexAppServerProtocolV2) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalCommandExecutionRequestApprovalParams(data []byte) (CommandExecutionRequestApprovalParams, error) {
	var r CommandExecutionRequestApprovalParams
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *CommandExecutionRequestApprovalParams) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalCommandExecutionRequestApprovalResponse(data []byte) (CommandExecutionRequestApprovalResponse, error) {
	var r CommandExecutionRequestApprovalResponse
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *CommandExecutionRequestApprovalResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalDynamicToolCallParams(data []byte) (DynamicToolCallParams, error) {
	var r DynamicToolCallParams
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *DynamicToolCallParams) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalDynamicToolCallResponse(data []byte) (DynamicToolCallResponse, error) {
	var r DynamicToolCallResponse
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *DynamicToolCallResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalExecCommandApprovalParams(data []byte) (ExecCommandApprovalParams, error) {
	var r ExecCommandApprovalParams
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ExecCommandApprovalParams) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalExecCommandApprovalResponse(data []byte) (ExecCommandApprovalResponse, error) {
	var r ExecCommandApprovalResponse
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ExecCommandApprovalResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalFileChangeRequestApprovalParams(data []byte) (FileChangeRequestApprovalParams, error) {
	var r FileChangeRequestApprovalParams
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *FileChangeRequestApprovalParams) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalFileChangeRequestApprovalResponse(data []byte) (FileChangeRequestApprovalResponse, error) {
	var r FileChangeRequestApprovalResponse
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *FileChangeRequestApprovalResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalFuzzyFileSearchParams(data []byte) (FuzzyFileSearchParams, error) {
	var r FuzzyFileSearchParams
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *FuzzyFileSearchParams) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalFuzzyFileSearchResponse(data []byte) (FuzzyFileSearchResponse, error) {
	var r FuzzyFileSearchResponse
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *FuzzyFileSearchResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalFuzzyFileSearchSessionCompletedNotification(data []byte) (FuzzyFileSearchSessionCompletedNotification, error) {
	var r FuzzyFileSearchSessionCompletedNotification
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *FuzzyFileSearchSessionCompletedNotification) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalFuzzyFileSearchSessionUpdatedNotification(data []byte) (FuzzyFileSearchSessionUpdatedNotification, error) {
	var r FuzzyFileSearchSessionUpdatedNotification
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *FuzzyFileSearchSessionUpdatedNotification) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalJSONRPCError(data []byte) (JSONRPCError, error) {
	var r JSONRPCError
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *JSONRPCError) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalJSONRPCErrorError(data []byte) (JSONRPCErrorError, error) {
	var r JSONRPCErrorError
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *JSONRPCErrorError) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalJSONRPCMessage(data []byte) (JSONRPCMessage, error) {
	var r JSONRPCMessage
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *JSONRPCMessage) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalJSONRPCNotification(data []byte) (JSONRPCNotification, error) {
	var r JSONRPCNotification
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *JSONRPCNotification) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalJSONRPCRequest(data []byte) (JSONRPCRequest, error) {
	var r JSONRPCRequest
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *JSONRPCRequest) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalJSONRPCResponse(data []byte) (JSONRPCResponse, error) {
	var r JSONRPCResponse
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *JSONRPCResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalMCPServerElicitationRequestParams(data []byte) (MCPServerElicitationRequestParams, error) {
	var r MCPServerElicitationRequestParams
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *MCPServerElicitationRequestParams) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalMCPServerElicitationRequestResponse(data []byte) (MCPServerElicitationRequestResponse, error) {
	var r MCPServerElicitationRequestResponse
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *MCPServerElicitationRequestResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalPermissionsRequestApprovalParams(data []byte) (PermissionsRequestApprovalParams, error) {
	var r PermissionsRequestApprovalParams
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *PermissionsRequestApprovalParams) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalPermissionsRequestApprovalResponse(data []byte) (PermissionsRequestApprovalResponse, error) {
	var r PermissionsRequestApprovalResponse
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *PermissionsRequestApprovalResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalRequestID(data []byte) (RequestID, error) {
	var r RequestID
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *RequestID) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalServerNotification(data []byte) (ServerNotification, error) {
	var r ServerNotification
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ServerNotification) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalServerRequest(data []byte) (ServerRequest, error) {
	var r ServerRequest
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ServerRequest) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalToolRequestUserInputParams(data []byte) (ToolRequestUserInputParams, error) {
	var r ToolRequestUserInputParams
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ToolRequestUserInputParams) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalToolRequestUserInputResponse(data []byte) (ToolRequestUserInputResponse, error) {
	var r ToolRequestUserInputResponse
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ToolRequestUserInputResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type ApplyPatchApprovalParams struct {
	// Use to correlate this with [codex_protocol::protocol::PatchApplyBeginEvent] and
	// [codex_protocol::protocol::PatchApplyEndEvent].
	CallID         string                                        `json:"callId"`
	ConversationID string                                        `json:"conversationId"`
	FileChanges    map[string]ApplyPatchApprovalParamsFileChange `json:"fileChanges"`
	// When set, the agent is asking the user to allow writes under this root for the remainder
	// of the session (unclear if this is honored today).
	GrantRoot *string `json:"grantRoot"`
	// Optional explanatory reason (e.g. request for extra write access).
	Reason *string `json:"reason"`
}

type ApplyPatchApprovalParamsFileChange struct {
	Content     *string `json:"content,omitempty"`
	Type        Type    `json:"type"`
	MovePath    *string `json:"move_path"`
	UnifiedDiff *string `json:"unified_diff,omitempty"`
}

type ApplyPatchApprovalResponse struct {
	Decision *ApplyPatchApprovalResponseReviewDecision `json:"decision"`
}

// User has approved this command and wants to apply the proposed execpolicy amendment so
// future matching commands are permitted.
//
// User chose to persist a network policy rule (allow/deny) for future requests to the same
// host.
//
// User has denied this command and the agent should not execute it, but it should continue
// the session and try something else.
type PurpleReviewDecision struct {
	ApprovedExecpolicyAmendment *PurpleApprovedExecpolicyAmendment `json:"approved_execpolicy_amendment,omitempty"`
	NetworkPolicyAmendment      *PurpleNetworkPolicyAmendment      `json:"network_policy_amendment,omitempty"`
	Denied                      *PurpleDenied                      `json:"denied,omitempty"`
}

type PurpleApprovedExecpolicyAmendment struct {
	ProposedExecpolicyAmendment []string `json:"proposed_execpolicy_amendment"`
}

type PurpleDenied struct {
	Rejection string `json:"rejection"`
}

type PurpleNetworkPolicyAmendment struct {
	NetworkPolicyAmendment FluffyNetworkPolicyAmendment `json:"network_policy_amendment"`
}

type FluffyNetworkPolicyAmendment struct {
	Action NetworkPolicyRuleAction `json:"action"`
	Host   string                  `json:"host"`
}

type AttestationGenerateResponse struct {
	// Opaque client attestation token.
	Token string `json:"token"`
}

type ChatgptAuthTokensRefreshParams struct {
	// Workspace/account identifier that Codex was previously using.
	//
	// Clients that manage multiple accounts/workspaces can use this as a hint to refresh the
	// token for the correct workspace.
	//
	// This may be `null` when the prior auth state did not include a workspace identifier
	// (`chatgpt_account_id`).
	PreviousAccountID *string                        `json:"previousAccountId"`
	Reason            ChatgptAuthTokensRefreshReason `json:"reason"`
}

type ChatgptAuthTokensRefreshResponse struct {
	AccessToken      string  `json:"accessToken"`
	ChatgptAccountID string  `json:"chatgptAccountId"`
	ChatgptPlanType  *string `json:"chatgptPlanType"`
}

type ClientNotification struct {
	Method InitializedNotificationMethod `json:"method"`
}

// Request from the client to the server.
//
// # NEW APIs
//
// Append raw Responses API items to the thread history without starting a user turn.
//
// Execute a standalone command (argv vector) under the server's sandbox.
//
// Write stdin bytes to a running `command/exec` session or close stdin.
//
// Terminate a running `command/exec` session by client-supplied `processId`.
//
// Resize a running PTY-backed `command/exec` session by client-supplied `processId`.
type ClientRequest struct {
	ID     *RequestID          `json:"id"`
	Method ClientRequestMethod `json:"method"`
	Params *Params             `json:"params"`
}

// There are three ways to resume a thread: 1. By thread_id: load the thread from disk by
// thread_id and resume it. 2. By history: instantiate the thread from memory and resume it.
// 3. By path: load the thread from disk by path and resume it.
//
// For non-running threads, the precedence is: history > non-empty path > thread_id. If
// using history or a non-empty path for a non-running thread, the thread_id param will be
// ignored.
//
// If thread_id identifies a running thread, app-server rejoins that thread and treats a
// non-empty path as a consistency check against the active rollout path. Empty string path
// values are treated as absent.
//
// Prefer using thread_id whenever possible.
//
// There are two ways to fork a thread: 1. By thread_id: load the thread from disk by
// thread_id and fork it into a new thread. 2. By path: load the thread from disk by path
// and fork it into a new thread.
//
// If using a non-empty path, the thread_id param will be ignored. Empty string path values
// are treated as absent.
//
// Prefer using thread_id whenever possible.
//
// Parameters for moving a thread within a server-owned section ordering.
//
// DEPRECATED: `thread/rollback` will be removed soon.
//
// Parameters for listing independently persisted thread sections.
//
// Parameters for creating an independently persisted thread section.
//
// Parameters for updating an independently persisted thread section.
//
// Parameters for deleting an independently persisted thread section.
//
// EXPERIMENTAL - read metadata for specific apps/connectors.
//
// EXPERIMENTAL - list available apps/connectors.
//
// Read the committed installed connector runtime snapshot.
//
// Read a file from the host filesystem.
//
// Write a file on the host filesystem.
//
// Create a directory on the host filesystem.
//
// Request metadata for an absolute path.
//
// List direct child names for a directory.
//
// Remove a file or directory tree from the host filesystem.
//
// Copy a file or directory tree on the host filesystem.
//
// Start filesystem watch notifications for an absolute path.
//
// Stop filesystem watch notifications for a prior `fs/watch`.
//
// [UNSTABLE] FOR OPENAI INTERNAL USE ONLY - DO NOT USE. The access token must contain the
// same scopes that Codex-managed ChatGPT auth tokens have.
//
// [UNSTABLE] Managed Amazon Bedrock login is experimental.
//
// Run a standalone command (argv vector) in the server sandbox without creating a thread or
// turn.
//
// The final `command/exec` response is deferred until the process exits and is sent only
// after all `command/exec/outputDelta` notifications for that connection have been
// emitted.
//
// Write stdin bytes to a running `command/exec` session, close stdin, or both.
//
// Terminate a running `command/exec` session.
//
// Resize a running PTY-backed `command/exec` session.
type Params struct {
	Capabilities *InitializeCapabilities `json:"capabilities"`
	ClientInfo   *ClientInfo             `json:"clientInfo,omitempty"`
	// Override the approval policy for this turn and subsequent turns.
	ApprovalPolicy *ApprovalPolicyUnion `json:"approvalPolicy"`
	// Override where approval requests are routed for review on this thread and subsequent
	// turns.
	//
	// Override where approval requests are routed for review on this turn and subsequent turns.
	ApprovalsReviewer *ApprovalsReviewer     `json:"approvalsReviewer"`
	BaseInstructions  *string                `json:"baseInstructions"`
	Config            map[string]interface{} `json:"config"`
	// Optional cwd filter or filters; when set, only threads whose session cwd exactly matches
	// one of these paths are returned.
	//
	// Override the working directory for this turn and subsequent turns.
	//
	// Optional working directory to resolve project config layers.
	//
	// Optional working directory. Defaults to the server cwd.
	//
	// Optional working directory to resolve project config layers. If specified, return the
	// effective config as seen from that directory (i.e., including any project layers between
	// `cwd` and the project/repo root).
	Cwd                   *ThreadListCwdFilter `json:"cwd"`
	DeveloperInstructions *string              `json:"developerInstructions"`
	Ephemeral             *bool                `json:"ephemeral"`
	// Configuration overrides for the resumed thread, if any.
	//
	// Configuration overrides for the forked thread, if any.
	//
	// Override the model for this turn and subsequent turns.
	Model         *string `json:"model"`
	ModelProvider *string `json:"modelProvider"`
	// Override the personality for this turn and subsequent turns.
	Personality *Personality `json:"personality"`
	Sandbox     *SandboxMode `json:"sandbox"`
	ServiceName *string      `json:"serviceName"`
	// Override the service tier for this turn and subsequent turns.
	ServiceTier        *string            `json:"serviceTier"`
	SessionStartSource *ThreadStartSource `json:"sessionStartSource"`
	// Optional client-supplied analytics source classification for this thread.
	//
	// Optional client-supplied analytics source classification for this forked thread.
	ThreadSource *string `json:"threadSource"`
	// Thread to move into, within, or out of a section.
	//
	// Optional loaded thread id used to evaluate effective app configuration.
	//
	// Optional thread id used to evaluate app feature gating from that thread's config.
	//
	// Optional loaded thread id. Pass this when showing feature state for an existing thread so
	// enablement is computed from that thread's refreshed config, including project-local
	// config for the thread's cwd.
	//
	// When present, read estimated usage for this thread instead of account-wide token activity.
	ThreadID *string `json:"threadId"`
	// Optional last turn id to fork through, inclusive.
	//
	// When specified, turns after `last_turn_id` are omitted from the fork. The referenced turn
	// cannot be in progress.
	LastTurnID *string `json:"lastTurnId"`
	// The user-visible name of the section.
	//
	// The updated user-visible name of the section.
	//
	// Name-based selector.
	Name        *string           `json:"name"`
	Objective   *string           `json:"objective"`
	Status      *ThreadGoalStatus `json:"status"`
	TokenBudget *int64            `json:"tokenBudget"`
	// Patch the stored Git metadata for this thread. Omit a field to leave it unchanged, set it
	// to `null` to clear it, or provide a string to replace the stored value.
	GitInfo *ThreadMetadataGitInfoUpdateParams `json:"gitInfo"`
	// Existing thread to insert before; omission or null appends to the section.
	BeforeThreadID *string `json:"beforeThreadId"`
	// Destination section, or `null` to remove the thread from its section.
	//
	// Omit to include every section, set to `null` for unsectioned threads, or provide a
	// section ID to return only threads in that section.
	//
	// The stable, server-generated identity of the section to update.
	//
	// The stable, server-generated identity of the section to delete.
	SectionID *string `json:"sectionId"`
	// Shell command string evaluated by the thread's configured shell. Unlike `command/exec`,
	// this intentionally preserves shell syntax such as pipes, redirects, and quoting. This
	// runs unsandboxed with full access rather than inheriting the thread sandbox policy.
	//
	// Command argv vector. Empty arrays are rejected.
	Command *Command `json:"command"`
	// Serialized `codex_protocol::protocol::GuardianAssessmentEvent`.
	Event interface{} `json:"event"`
	// The number of turns to drop from the end of the thread. Must be >= 1.
	//
	// This only modifies the thread's history and does not revert local file changes that have
	// been made by the agent. Clients are responsible for reverting these changes.
	NumTurns *int64 `json:"numTurns,omitempty"`
	// Optional archived filter; when set to true, only archived threads are returned. If false
	// or null, only non-archived threads are returned.
	Archived *bool `json:"archived"`
	// Opaque pagination cursor returned by a previous call.
	Cursor *string `json:"cursor"`
	// Optional page size; defaults to a reasonable server-side value.
	//
	// Maximum number of sections to return.
	//
	// Optional page size; defaults to no limit.
	//
	// Optional page size; defaults to the full result set.
	//
	// Optional page size; defaults to a server-defined value.
	Limit *int64 `json:"limit"`
	// Optional provider filter; when set, only sessions recorded under these providers are
	// returned. When present but empty, includes all providers.
	ModelProviders []string `json:"modelProviders"`
	// Optional substring filter for the extracted thread title.
	SearchTerm *string `json:"searchTerm"`
	// Optional sort direction; defaults to descending (newest first).
	SortDirection *SortDirection `json:"sortDirection"`
	// Optional sort key; defaults to created_at.
	SortKey *ThreadSortKey `json:"sortKey"`
	// Optional source filter; when set, only sessions from these source kinds are returned.
	// When omitted or empty, defaults to interactive sources.
	SourceKinds []ThreadSourceKind `json:"sourceKinds"`
	// If true, return from the state DB without scanning JSONL rollouts to repair thread
	// metadata. Omitted or false preserves scan-and-repair behavior.
	UseStateDBOnly *bool `json:"useStateDbOnly,omitempty"`
	// Omit to preserve appearance, use `null` to clear it, or provide a replacement.
	Appearance *InitializeParamsThreadSectionAppearance `json:"appearance"`
	// When true, include turns and their items from rollout history.
	IncludeTurns *bool `json:"includeTurns,omitempty"`
	// Raw Responses API items to append to the thread's model-visible history.
	Items []interface{} `json:"items,omitempty"`
	// When empty, defaults to the current session working directory.
	//
	// Optional working directories used to discover repo marketplaces. When omitted, only
	// home-scoped marketplaces and the official curated marketplace are considered.
	//
	// Optional working directories used to discover repo marketplaces.
	//
	// Zero or more working directories to include for repo-scoped detection.
	Cwds []string `json:"cwds"`
	// When true, bypass the skills cache and re-scan skills from disk.
	ForceReload *bool    `json:"forceReload,omitempty"`
	ExtraRoots  []string `json:"extraRoots,omitempty"`
	RefName     *string  `json:"refName"`
	// Deprecated field retained for compatibility. This field is ignored; use `migrationSource`
	// to select the migration source.
	//
	// Optional identifier for the product that initiated the import.
	Source          *string  `json:"source"`
	SparsePaths     []string `json:"sparsePaths"`
	MarketplaceName *string  `json:"marketplaceName"`
	// Whether the client requests a fresh remote plugin catalog fetch.
	//
	// When true, bypass app caches and fetch the latest data from sources.
	ForceRefetch *bool `json:"forceRefetch,omitempty"`
	// Optional marketplace kind filter. When omitted, only local marketplaces are queried, plus
	// the default remote catalog when enabled by feature flag.
	MarketplaceKinds []PluginListMarketplaceKind `json:"marketplaceKinds"`
	// Additional uninstalled plugin names that should be returned when present locally. This is
	// used by mention surfaces that intentionally expose install entrypoints.
	InstallSuggestionPluginNames []string                     `json:"installSuggestionPluginNames"`
	MarketplacePath              *string                      `json:"marketplacePath"`
	PluginName                   *string                      `json:"pluginName,omitempty"`
	RemoteMarketplaceName        *string                      `json:"remoteMarketplaceName"`
	RemotePluginID               *string                      `json:"remotePluginId"`
	SkillName                    *string                      `json:"skillName,omitempty"`
	Discoverability              *PluginShareEDiscoverability `json:"discoverability"`
	PluginPath                   *string                      `json:"pluginPath,omitempty"`
	ShareTargets                 []PluginShareTarget          `json:"shareTargets"`
	// App ids to read. The server accepts at most 100 ids and deduplicates repeated ids while
	// preserving their first-request order.
	AppIDS []string `json:"appIds,omitempty"`
	// When true, include display-only public tool summaries in the returned metadata.
	IncludeTools *bool `json:"includeTools,omitempty"`
	// When true and Apps are permitted, refresh and publish the hosted connector runtime tool
	// snapshot first.
	ForceRefresh *bool `json:"forceRefresh,omitempty"`
	// Absolute path to read.
	//
	// Absolute path to write.
	//
	// Absolute directory path to create.
	//
	// Absolute path to inspect.
	//
	// Absolute directory path to read.
	//
	// Absolute path to remove.
	//
	// Absolute file or directory path to watch.
	//
	// Path-based selector.
	Path *string `json:"path"`
	// File contents encoded as base64.
	DataBase64 *string `json:"dataBase64,omitempty"`
	// Whether parent directories should also be created. Defaults to `true`.
	//
	// Whether directory removal should recurse. Defaults to `true`.
	//
	// Required for directory copies; ignored for file copies.
	Recursive *bool `json:"recursive"`
	// Whether missing paths should be ignored. Defaults to `true`.
	Force *bool `json:"force"`
	// Absolute destination path.
	DestinationPath *string `json:"destinationPath,omitempty"`
	// Absolute source path.
	SourcePath *string `json:"sourcePath,omitempty"`
	// Connection-scoped watch identifier used for `fs/unwatch` and `fs/changed`.
	//
	// Watch identifier previously provided to `fs/watch`.
	WatchID *string `json:"watchId,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
	// Client-generated identifier used to correlate one installation attempt.
	InstallAttemptID    *string `json:"installAttemptId"`
	PluginID            *string `json:"pluginId,omitempty"`
	ClientUserMessageID *string `json:"clientUserMessageId"`
	// Override the reasoning effort for this turn and subsequent turns.
	Effort *string     `json:"effort"`
	Input  []UserInput `json:"input,omitempty"`
	// Optional JSON Schema used to constrain the final assistant message for this turn.
	OutputSchema interface{} `json:"outputSchema"`
	// Override the sandbox policy for this turn and subsequent turns.
	//
	// Optional sandbox policy for this command.
	//
	// Uses the same shape as thread/turn execution sandbox configuration and defaults to the
	// user's configured policy when omitted. Cannot be combined with `permissionProfile`.
	SandboxPolicy *SandboxPolicy `json:"sandboxPolicy"`
	// Override the reasoning summary for this turn and subsequent turns.
	Summary *ReasoningSummary `json:"summary"`
	// Required active turn id precondition. The request fails when it does not match the
	// currently active turn.
	ExpectedTurnID *string `json:"expectedTurnId,omitempty"`
	TurnID         *string `json:"turnId,omitempty"`
	// Where to run the review: inline (default) on the current thread or detached on a new
	// thread (returned in `reviewThreadId`).
	Delivery *ReviewDelivery `json:"delivery"`
	Target   *ReviewTarget   `json:"target,omitempty"`
	// When true, include models that are hidden from the default picker list.
	IncludeHidden *bool `json:"includeHidden"`
	// Process-wide runtime feature enablement keyed by canonical feature name.
	//
	// Only named features are updated. Omitted features are left unchanged. Send an empty map
	// for a no-op.
	Enablement map[string]bool `json:"enablement,omitempty"`
	// Registration strategy for this login only; omission selects automatic discovery.
	ClientRegistration *MCPServerOauthClientRegistration `json:"clientRegistration"`
	Scopes             []string                          `json:"scopes"`
	TimeoutSecs        *int64                            `json:"timeoutSecs"`
	// Controls how much MCP inventory data to fetch for each server. Defaults to `Full` when
	// omitted.
	Detail      *MCPServerStatusDetail `json:"detail"`
	ConnectorID *string                `json:"connectorId"`
	// Originating MCP tool call used to select the resource's app.
	OriginCallID              *string                  `json:"originCallId"`
	Server                    *string                  `json:"server,omitempty"`
	URI                       *string                  `json:"uri,omitempty"`
	Meta                      interface{}              `json:"_meta"`
	Arguments                 interface{}              `json:"arguments"`
	Tool                      *string                  `json:"tool,omitempty"`
	Mode                      *WindowsSandboxSetupMode `json:"mode,omitempty"`
	APIKey                    *string                  `json:"apiKey,omitempty"`
	Type                      *LoginAccountParamsType  `json:"type,omitempty"`
	AppBrand                  *LoginAppBrand           `json:"appBrand"`
	CodexStreamlinedLogin     *bool                    `json:"codexStreamlinedLogin,omitempty"`
	UseHostedLoginSuccessPage *bool                    `json:"useHostedLoginSuccessPage,omitempty"`
	// Access token (JWT) supplied by the client. This token is used for backend API requests
	// and email extraction.
	AccessToken *string `json:"accessToken,omitempty"`
	// Workspace/account identifier supplied by the client.
	ChatgptAccountID *string `json:"chatgptAccountId,omitempty"`
	// Optional plan type supplied by the client.
	//
	// When `null`, Codex attempts to derive the plan type from access-token claims. If
	// unavailable, the plan defaults to `unknown`.
	ChatgptPlanType *string `json:"chatgptPlanType"`
	Region          *string `json:"region,omitempty"`
	LoginID         *string `json:"loginId,omitempty"`
	// Opaque reset-credit identifier to redeem. When omitted, the backend selects the next
	// available credit.
	CreditID *string `json:"creditId"`
	// Identifies one logical reset attempt. A UUID is recommended; reuse the same value when
	// retrying that attempt.
	IdempotencyKey *string                    `json:"idempotencyKey,omitempty"`
	CreditType     *AddCreditsNudgeCreditType `json:"creditType,omitempty"`
	Classification *string                    `json:"classification,omitempty"`
	ExtraLogFiles  []string                   `json:"extraLogFiles"`
	IncludeLogs    *bool                      `json:"includeLogs,omitempty"`
	Reason         *string                    `json:"reason"`
	Tags           map[string]string          `json:"tags"`
	// Disable stdout/stderr capture truncation for this request.
	//
	// Cannot be combined with `outputBytesCap`.
	DisableOutputCap *bool `json:"disableOutputCap,omitempty"`
	// Disable the timeout entirely for this request.
	//
	// Cannot be combined with `timeoutMs`.
	DisableTimeout *bool `json:"disableTimeout,omitempty"`
	// Optional environment overrides merged into the server-computed environment.
	//
	// Matching names override inherited values. Set a key to `null` to unset an inherited
	// variable.
	Env map[string]*string `json:"env"`
	// Optional per-stream stdout/stderr capture cap in bytes.
	//
	// When omitted, the server default applies. Cannot be combined with `disableOutputCap`.
	OutputBytesCap *int64 `json:"outputBytesCap"`
	// Optional client-supplied, connection-scoped process id.
	//
	// Required for `tty`, `streamStdin`, `streamStdoutStderr`, and follow-up
	// `command/exec/write`, `command/exec/resize`, and `command/exec/terminate` calls. When
	// omitted, buffered execution gets an internal id that is not exposed to the client.
	//
	// Client-supplied, connection-scoped `processId` from the original `command/exec` request.
	ProcessID *string `json:"processId"`
	// Optional initial PTY size in character cells. Only valid when `tty` is true.
	//
	// New PTY size in character cells.
	Size *CommandExecTerminalSize `json:"size"`
	// Allow follow-up `command/exec/write` requests to write stdin bytes.
	//
	// Requires a client-supplied `processId`.
	StreamStdin *bool `json:"streamStdin,omitempty"`
	// Stream stdout/stderr via `command/exec/outputDelta` notifications.
	//
	// Streamed bytes are not duplicated into the final response and require a client-supplied
	// `processId`.
	StreamStdoutStderr *bool `json:"streamStdoutStderr,omitempty"`
	// Optional timeout in milliseconds.
	//
	// When omitted, the server default applies. Cannot be combined with `disableTimeout`.
	TimeoutMS *int64 `json:"timeoutMs"`
	// Enable PTY mode.
	//
	// This implies `streamStdin` and `streamStdoutStderr`.
	TTY *bool `json:"tty,omitempty"`
	// Close stdin after writing `deltaBase64`, if present.
	CloseStdin *bool `json:"closeStdin,omitempty"`
	// Optional base64-encoded stdin bytes to write.
	DeltaBase64   *string `json:"deltaBase64"`
	IncludeLayers *bool   `json:"includeLayers,omitempty"`
	// If true, include detection under the user's home directory.
	IncludeHome *bool `json:"includeHome,omitempty"`
	// Maximum age in days for detected sessions. Missing values use the default limit.
	MaxSessionAgeDays *int64 `json:"maxSessionAgeDays"`
	// Maximum number of sessions to detect. Missing values use the default limit.
	MaxSessions *int64 `json:"maxSessions"`
	// Optional migration-source selector. Missing or unrecognized values use the default
	// source.
	//
	// Migration-source selector used to produce the migration items. Pass the same value to
	// detection and import; missing or unrecognized values use the default source.
	MigrationSource *string                            `json:"migrationSource"`
	MigrationItems  []ExternalAgentConfigMigrationItem `json:"migrationItems,omitempty"`
	// Opaque provider identifier supplied by the caller for analytics attribution and import
	// history display. This does not select the migration source.
	//
	// Opaque provider identifier for the externally completed import.
	ProviderID *string `json:"providerId"`
	// Completed results grouped by imported item type.
	ItemTypeResults []ExternalAgentConfigImportHistoryRecordTypeResultParams `json:"itemTypeResults,omitempty"`
	ExpectedVersion *string                                                  `json:"expectedVersion"`
	// Path to the config file to write; defaults to the user's `config.toml` when omitted.
	FilePath      *string        `json:"filePath"`
	KeyPath       *string        `json:"keyPath,omitempty"`
	MergeStrategy *MergeStrategy `json:"mergeStrategy,omitempty"`
	Value         interface{}    `json:"value"`
	Edits         []ConfigEdit   `json:"edits,omitempty"`
	// When true, hot-reload updated runtime settings into loaded threads after writing.
	// Session-static model, reasoning-effort, Plan-mode reasoning-effort, service-tier, and
	// personality defaults are not reloaded.
	ReloadUserConfig *bool `json:"reloadUserConfig,omitempty"`
	// When `true`, requests a proactive token refresh before returning.
	//
	// In managed auth mode this triggers the normal refresh-token flow. In external auth mode
	// this flag is ignored. Clients should refresh tokens themselves and call
	// `account/login/start` with `chatgptAuthTokens`.
	RefreshToken      *bool    `json:"refreshToken,omitempty"`
	CancellationToken *string  `json:"cancellationToken"`
	Query             *string  `json:"query,omitempty"`
	Roots             []string `json:"roots,omitempty"`
}

// Extensible visual presentation for a custom thread section.
type InitializeParamsThreadSectionAppearance struct {
	Color *string `json:"color"`
	Icon  *string `json:"icon"`
}

type ApprovalPolicyGranularAskForApproval struct {
	Granular PurpleGranular `json:"granular"`
}

type PurpleGranular struct {
	MCPElicitations    bool  `json:"mcp_elicitations"`
	RequestPermissions *bool `json:"request_permissions,omitempty"`
	Rules              bool  `json:"rules"`
	SandboxApproval    bool  `json:"sandbox_approval"`
	SkillApproval      *bool `json:"skill_approval,omitempty"`
}

// Client-declared capabilities negotiated during initialize.
type InitializeCapabilities struct {
	// Opt into receiving experimental API methods and fields.
	ExperimentalAPI *bool `json:"experimentalApi,omitempty"`
	// MCP extension settings declared by the app-server client.
	Extensions map[string]interface{} `json:"extensions"`
	// Legacy opt-in for the `openai/form` MCP extension.
	//
	// New clients should declare `openai/form` in [`Self::extensions`].
	MCPServerOpenaiFormElicitation *bool `json:"mcpServerOpenaiFormElicitation,omitempty"`
	// Exact notification method names that should be suppressed for this connection (for
	// example `thread/started`).
	OptOutNotificationMethods []string `json:"optOutNotificationMethods"`
	// Opt into `attestation/generate` requests for upstream `x-oai-attestation`.
	RequestAttestation *bool `json:"requestAttestation,omitempty"`
}

type ClientInfo struct {
	Name    string  `json:"name"`
	Title   *string `json:"title"`
	Version string  `json:"version"`
}

type ConfigEdit struct {
	KeyPath       string        `json:"keyPath"`
	MergeStrategy MergeStrategy `json:"mergeStrategy"`
	Value         interface{}   `json:"value"`
}

type ThreadMetadataGitInfoUpdateParams struct {
	// Omit to leave the stored branch unchanged, set to `null` to clear it, or provide a
	// non-empty string to replace it.
	Branch *string `json:"branch"`
	// Omit to leave the stored origin URL unchanged, set to `null` to clear it, or provide a
	// non-empty string to replace it.
	OriginURL *string `json:"originUrl"`
	// Omit to leave the stored commit unchanged, set to `null` to clear it, or provide a
	// non-empty string to replace it.
	SHA *string `json:"sha"`
}

type UserInput struct {
	Text *string `json:"text,omitempty"`
	// UI-defined spans within `text` used to render or persist special elements.
	TextElements []UserInputTextElement `json:"text_elements,omitempty"`
	Type         UserInputType          `json:"type"`
	Detail       *ImageDetail           `json:"detail"`
	URL          *string                `json:"url,omitempty"`
	Path         *string                `json:"path,omitempty"`
	Name         *string                `json:"name,omitempty"`
}

type UserInputTextElement struct {
	// Byte range in the parent `text` buffer that this element occupies.
	ByteRange PurpleByteRange `json:"byteRange"`
	// Optional human-readable placeholder for the element, displayed in the UI.
	Placeholder *string `json:"placeholder"`
}

// Byte range in the parent `text` buffer that this element occupies.
type PurpleByteRange struct {
	End   int64 `json:"end"`
	Start int64 `json:"start"`
}

type ExternalAgentConfigImportHistoryRecordTypeResultParams struct {
	Failures  []PurpleExternalAgentConfigImportItemTypeFailure      `json:"failures"`
	ItemType  ExternalAgentConfigMigrationItemType                  `json:"itemType"`
	Successes []ExternalAgentConfigImportHistoryRecordSuccessParams `json:"successes"`
}

type PurpleExternalAgentConfigImportItemTypeFailure struct {
	Cwd          *string                              `json:"cwd"`
	ErrorType    *string                              `json:"errorType"`
	FailureStage string                               `json:"failureStage"`
	ItemType     ExternalAgentConfigMigrationItemType `json:"itemType"`
	Message      string                               `json:"message"`
	Source       *string                              `json:"source"`
	SubErrorType *string                              `json:"subErrorType"`
}

type ExternalAgentConfigImportHistoryRecordSuccessParams struct {
	Cwd      *string                              `json:"cwd"`
	ItemType ExternalAgentConfigMigrationItemType `json:"itemType"`
	Source   *string                              `json:"source"`
	Target   *string                              `json:"target"`
	// Original title for an imported session, when available.
	Title *string `json:"title"`
}

type ExternalAgentConfigMigrationItem struct {
	// Null or empty means home-scoped migration; non-empty means repo-scoped migration.
	Cwd         *string                              `json:"cwd"`
	Description string                               `json:"description"`
	Details     *MigrationDetails                    `json:"details"`
	ItemType    ExternalAgentConfigMigrationItemType `json:"itemType"`
}

type MigrationDetails struct {
	Commands   []CommandMigration   `json:"commands,omitempty"`
	Hooks      []HookMigration      `json:"hooks,omitempty"`
	MCPServers []MCPServerMigration `json:"mcpServers,omitempty"`
	Memory     []string             `json:"memory,omitempty"`
	Plugins    []PluginsMigration   `json:"plugins,omitempty"`
	Sessions   []SessionMigration   `json:"sessions,omitempty"`
	Skills     []SkillMigration     `json:"skills,omitempty"`
	Subagents  []SubagentMigration  `json:"subagents,omitempty"`
}

type CommandMigration struct {
	Name string `json:"name"`
}

type HookMigration struct {
	Name string `json:"name"`
}

type MCPServerMigration struct {
	Name string `json:"name"`
}

type PluginsMigration struct {
	MarketplaceName string   `json:"marketplaceName"`
	PluginNames     []string `json:"pluginNames"`
}

type SessionMigration struct {
	Cwd   string  `json:"cwd"`
	Path  string  `json:"path"`
	Title *string `json:"title"`
}

type SkillMigration struct {
	Name string `json:"name"`
}

type SubagentMigration struct {
	Name string `json:"name"`
}

type SandboxPolicy struct {
	Type                SandboxPolicyType   `json:"type"`
	NetworkAccess       *NetworkAccessUnion `json:"networkAccess"`
	ExcludeSlashTmp     *bool               `json:"excludeSlashTmp,omitempty"`
	ExcludeTmpdirEnvVar *bool               `json:"excludeTmpdirEnvVar,omitempty"`
	WritableRoots       []string            `json:"writableRoots,omitempty"`
}

type PluginShareTarget struct {
	PrincipalID   string                   `json:"principalId"`
	PrincipalType PluginSharePrincipalType `json:"principalType"`
	Role          PluginShareTargetRole    `json:"role"`
}

// PTY size in character cells for `command/exec` PTY sessions.
//
// New PTY size in character cells.
type CommandExecTerminalSize struct {
	// Terminal width in character cells.
	Cols int64 `json:"cols"`
	// Terminal height in character cells.
	Rows int64 `json:"rows"`
}

// Review the working tree: staged, unstaged, and untracked files.
//
// Review changes between the current branch and the given base branch.
//
// Review the changes introduced by a specific commit.
//
// Arbitrary instructions, equivalent to the old free-form prompt.
type ReviewTarget struct {
	Type   ReviewTargetType `json:"type"`
	Branch *string          `json:"branch,omitempty"`
	SHA    *string          `json:"sha,omitempty"`
	// Optional human-readable label (e.g., commit subject) for UIs.
	Title        *string `json:"title"`
	Instructions *string `json:"instructions,omitempty"`
}

type CommandExecutionRequestApprovalParams struct {
	// Unique identifier for this specific approval callback.
	//
	// For regular shell/unified_exec approvals, this is null.
	//
	// For zsh-exec-bridge subcommand approvals, multiple callbacks can belong to one parent
	// `itemId`, so `approvalId` is a distinct opaque callback id (a UUID) used to disambiguate
	// routing.
	ApprovalID *string `json:"approvalId"`
	// The command to be executed.
	Command *string `json:"command"`
	// Best-effort parsed command actions for friendly display.
	CommandActions []CommandExecutionRequestApprovalParamsCommandAction `json:"commandActions"`
	// The command's working directory.
	Cwd *string `json:"cwd"`
	// Environment in which the command will run.
	EnvironmentID *string `json:"environmentId"`
	ItemID        string  `json:"itemId"`
	// Optional context for a managed-network approval prompt.
	NetworkApprovalContext *CommandExecutionRequestApprovalParamsNetworkApprovalContext `json:"networkApprovalContext"`
	// Optional proposed execpolicy amendment to allow similar commands without prompting.
	ProposedExecpolicyAmendment []string `json:"proposedExecpolicyAmendment"`
	// Optional proposed network policy amendments (allow/deny host) for future requests.
	ProposedNetworkPolicyAmendments []CommandExecutionRequestApprovalParamsProposedNetworkPolicyAmendment `json:"proposedNetworkPolicyAmendments"`
	// Optional explanatory reason (e.g. request for network access).
	Reason *string `json:"reason"`
	// Unix timestamp (in milliseconds) when this approval request started.
	StartedAtMS int64  `json:"startedAtMs"`
	ThreadID    string `json:"threadId"`
	TurnID      string `json:"turnId"`
}

type CommandExecutionRequestApprovalParamsCommandAction struct {
	Command string            `json:"command"`
	Name    *string           `json:"name,omitempty"`
	Path    *string           `json:"path"`
	Type    CommandActionType `json:"type"`
	Query   *string           `json:"query"`
}

type CommandExecutionRequestApprovalParamsNetworkApprovalContext struct {
	Host     string                  `json:"host"`
	Protocol NetworkApprovalProtocol `json:"protocol"`
}

type CommandExecutionRequestApprovalParamsProposedNetworkPolicyAmendment struct {
	Action NetworkPolicyRuleAction `json:"action"`
	Host   string                  `json:"host"`
}

type CommandExecutionRequestApprovalResponse struct {
	Decision *CommandExecutionApprovalDecision `json:"decision"`
}

// User approved the command, and wants to apply the proposed execpolicy amendment so future
// matching commands can run without prompting.
//
// User chose a persistent network policy rule (allow/deny) for this host.
type PolicyAmendmentCommandExecutionApprovalDecision struct {
	AcceptWithExecpolicyAmendment *AcceptWithExecpolicyAmendment `json:"acceptWithExecpolicyAmendment,omitempty"`
	ApplyNetworkPolicyAmendment   *ApplyNetworkPolicyAmendment   `json:"applyNetworkPolicyAmendment,omitempty"`
}

type AcceptWithExecpolicyAmendment struct {
	ExecpolicyAmendment []string `json:"execpolicy_amendment"`
}

type ApplyNetworkPolicyAmendment struct {
	NetworkPolicyAmendment ApplyNetworkPolicyAmendmentNetworkPolicyAmendment `json:"network_policy_amendment"`
}

type ApplyNetworkPolicyAmendmentNetworkPolicyAmendment struct {
	Action NetworkPolicyRuleAction `json:"action"`
	Host   string                  `json:"host"`
}

type DynamicToolCallParams struct {
	Arguments interface{} `json:"arguments"`
	CallID    string      `json:"callId"`
	Namespace *string     `json:"namespace"`
	ThreadID  string      `json:"threadId"`
	Tool      string      `json:"tool"`
	TurnID    string      `json:"turnId"`
}

type DynamicToolCallResponse struct {
	ContentItems []DynamicToolCallResponseDynamicToolCallOutputContentItem `json:"contentItems"`
	Success      bool                                                      `json:"success"`
}

type DynamicToolCallResponseDynamicToolCallOutputContentItem struct {
	Text     *string                                   `json:"text,omitempty"`
	Type     InputDynamicToolCallOutputContentItemType `json:"type"`
	ImageURL *string                                   `json:"imageUrl,omitempty"`
	AudioURL *string                                   `json:"audioUrl,omitempty"`
}

type ExecCommandApprovalParams struct {
	// Identifier for this specific approval callback.
	ApprovalID *string `json:"approvalId"`
	// Use to correlate this with [codex_protocol::protocol::ExecCommandBeginEvent] and
	// [codex_protocol::protocol::ExecCommandEndEvent].
	CallID         string                                   `json:"callId"`
	Command        []string                                 `json:"command"`
	ConversationID string                                   `json:"conversationId"`
	Cwd            string                                   `json:"cwd"`
	ParsedCmd      []ExecCommandApprovalParamsParsedCommand `json:"parsedCmd"`
	Reason         *string                                  `json:"reason"`
}

type ExecCommandApprovalParamsParsedCommand struct {
	Cmd  string  `json:"cmd"`
	Name *string `json:"name,omitempty"`
	// (Best effort) Path to the file being read by the command. When possible, this is an
	// absolute path, though when relative, it should be resolved against the `cwd`` that will
	// be used to run the command to derive the absolute path.
	Path  *string           `json:"path"`
	Type  ParsedCommandType `json:"type"`
	Query *string           `json:"query"`
}

type ExecCommandApprovalResponse struct {
	Decision *ExecCommandApprovalResponseReviewDecision `json:"decision"`
}

// User has approved this command and wants to apply the proposed execpolicy amendment so
// future matching commands are permitted.
//
// User chose to persist a network policy rule (allow/deny) for future requests to the same
// host.
//
// User has denied this command and the agent should not execute it, but it should continue
// the session and try something else.
type FluffyReviewDecision struct {
	ApprovedExecpolicyAmendment *FluffyApprovedExecpolicyAmendment `json:"approved_execpolicy_amendment,omitempty"`
	NetworkPolicyAmendment      *TentacledNetworkPolicyAmendment   `json:"network_policy_amendment,omitempty"`
	Denied                      *FluffyDenied                      `json:"denied,omitempty"`
}

type FluffyApprovedExecpolicyAmendment struct {
	ProposedExecpolicyAmendment []string `json:"proposed_execpolicy_amendment"`
}

type FluffyDenied struct {
	Rejection string `json:"rejection"`
}

type TentacledNetworkPolicyAmendment struct {
	NetworkPolicyAmendment StickyNetworkPolicyAmendment `json:"network_policy_amendment"`
}

type StickyNetworkPolicyAmendment struct {
	Action NetworkPolicyRuleAction `json:"action"`
	Host   string                  `json:"host"`
}

type FileChangeRequestApprovalParams struct {
	// [UNSTABLE] When set, the agent is asking the user to allow writes under this root for the
	// remainder of the session (unclear if this is honored today).
	GrantRoot *string `json:"grantRoot"`
	ItemID    string  `json:"itemId"`
	// Optional explanatory reason (e.g. request for extra write access).
	Reason *string `json:"reason"`
	// Unix timestamp (in milliseconds) when this approval request started.
	StartedAtMS int64  `json:"startedAtMs"`
	ThreadID    string `json:"threadId"`
	TurnID      string `json:"turnId"`
}

type FileChangeRequestApprovalResponse struct {
	Decision FileChangeApprovalDecision `json:"decision"`
}

type FuzzyFileSearchParams struct {
	CancellationToken *string  `json:"cancellationToken"`
	Query             string   `json:"query"`
	Roots             []string `json:"roots"`
}

type FuzzyFileSearchResponse struct {
	Files []FuzzyFileSearchResponseFile `json:"files"`
}

// Superset of [`codex_file_search::FileMatch`]
type FuzzyFileSearchResponseFile struct {
	FileName  string                   `json:"file_name"`
	Indices   []int64                  `json:"indices"`
	MatchType FuzzyFileSearchMatchType `json:"match_type"`
	Path      string                   `json:"path"`
	Root      string                   `json:"root"`
	Score     int64                    `json:"score"`
}

type FuzzyFileSearchSessionCompletedNotification struct {
	SessionID string `json:"sessionId"`
}

type FuzzyFileSearchSessionUpdatedNotification struct {
	Files     []FuzzyFileSearchSessionUpdatedNotificationFile `json:"files"`
	Query     string                                          `json:"query"`
	SessionID string                                          `json:"sessionId"`
}

// Superset of [`codex_file_search::FileMatch`]
type FuzzyFileSearchSessionUpdatedNotificationFile struct {
	FileName  string                   `json:"file_name"`
	Indices   []int64                  `json:"indices"`
	MatchType FuzzyFileSearchMatchType `json:"match_type"`
	Path      string                   `json:"path"`
	Root      string                   `json:"root"`
	Score     int64                    `json:"score"`
}

// A response to a request that indicates an error occurred.
type JSONRPCError struct {
	Error JSONRPCErrorErrorClass `json:"error"`
	ID    *RequestID             `json:"id"`
}

type JSONRPCErrorErrorClass struct {
	Code    int64       `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

type JSONRPCErrorError struct {
	Code    int64       `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

// Refers to any valid JSON-RPC object that can be decoded off the wire, or encoded to be
// sent.
//
// A request that expects a response.
//
// A notification which does not expect a response.
//
// A successful (non-error) response to a request.
//
// A response to a request that indicates an error occurred.
type JSONRPCMessage struct {
	ID     *RequestID  `json:"id"`
	Method *string     `json:"method,omitempty"`
	Params interface{} `json:"params"`
	// Optional W3C Trace Context for distributed tracing.
	Trace  *JSONRPCMessageW3CTraceContext `json:"trace"`
	Result interface{}                    `json:"result"`
	Error  *JSONRPCMessageError           `json:"error,omitempty"`
}

type JSONRPCMessageError struct {
	Code    int64       `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

type JSONRPCMessageW3CTraceContext struct {
	Traceparent *string `json:"traceparent"`
	Tracestate  *string `json:"tracestate"`
}

// A notification which does not expect a response.
type JSONRPCNotification struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

// A request that expects a response.
type JSONRPCRequest struct {
	ID     *RequestID  `json:"id"`
	Method string      `json:"method"`
	Params interface{} `json:"params"`
	// Optional W3C Trace Context for distributed tracing.
	Trace *JSONRPCRequestW3CTraceContext `json:"trace"`
}

type JSONRPCRequestW3CTraceContext struct {
	Traceparent *string `json:"traceparent"`
	Tracestate  *string `json:"tracestate"`
}

// A successful (non-error) response to a request.
type JSONRPCResponse struct {
	ID     *RequestID  `json:"id"`
	Result interface{} `json:"result"`
}

type MCPServerElicitationRequestParams struct {
	ServerName string `json:"serverName"`
	ThreadID   string `json:"threadId"`
	// Active Codex turn when this elicitation was observed, if app-server could correlate one.
	//
	// This is nullable because MCP models elicitation as a standalone server-to-client request
	// identified by the MCP server request id. It may be triggered during a turn, but turn
	// context is app-server correlation rather than part of the protocol identity of the
	// elicitation itself.
	TurnID          *string     `json:"turnId"`
	Meta            interface{} `json:"_meta"`
	Message         string      `json:"message"`
	Mode            Mode        `json:"mode"`
	RequestedSchema interface{} `json:"requestedSchema"`
	ElicitationID   *string     `json:"elicitationId,omitempty"`
	URL             *string     `json:"url,omitempty"`
}

type MCPServerElicitationRequestResponse struct {
	// Optional client metadata for form-mode action handling.
	Meta   interface{}                `json:"_meta"`
	Action MCPServerElicitationAction `json:"action"`
	// Structured user input for accepted elicitations, mirroring RMCP
	// `CreateElicitationResult`.
	//
	// This is nullable because decline/cancel responses have no content.
	Content interface{} `json:"content"`
}

type PermissionsRequestApprovalParams struct {
	Cwd           string                                      `json:"cwd"`
	EnvironmentID *string                                     `json:"environmentId"`
	ItemID        string                                      `json:"itemId"`
	Permissions   PermissionsRequestApprovalParamsPermissions `json:"permissions"`
	Reason        *string                                     `json:"reason"`
	// Unix timestamp (in milliseconds) when this approval request started.
	StartedAtMS int64  `json:"startedAtMs"`
	ThreadID    string `json:"threadId"`
	TurnID      string `json:"turnId"`
}

type PermissionsRequestApprovalParamsPermissions struct {
	FileSystem *PurpleAdditionalFileSystemPermissions `json:"fileSystem"`
	Network    *PurpleAdditionalNetworkPermissions    `json:"network"`
}

type PurpleAdditionalFileSystemPermissions struct {
	Entries          []PurpleFileSystemSandboxEntry `json:"entries"`
	GlobScanMaxDepth *int64                         `json:"globScanMaxDepth"`
	// This will be removed in favor of `entries`.
	Read []string `json:"read"`
	// This will be removed in favor of `entries`.
	Write []string `json:"write"`
}

type PurpleFileSystemSandboxEntry struct {
	Access FileSystemAccessMode `json:"access"`
	Path   PurpleFileSystemPath `json:"path"`
}

type PurpleFileSystemPath struct {
	Path    *string                      `json:"path,omitempty"`
	Type    FileSystemPathType           `json:"type"`
	Pattern *string                      `json:"pattern,omitempty"`
	Value   *PurpleFileSystemSpecialPath `json:"value,omitempty"`
}

type PurpleFileSystemSpecialPath struct {
	Kind    Kind    `json:"kind"`
	Subpath *string `json:"subpath"`
	Path    *string `json:"path,omitempty"`
}

type PurpleAdditionalNetworkPermissions struct {
	Enabled *bool `json:"enabled"`
}

type PermissionsRequestApprovalResponse struct {
	Permissions GrantedPermissionProfile `json:"permissions"`
	Scope       *PermissionGrantScope    `json:"scope,omitempty"`
	// Review every subsequent command in this turn before normal sandboxed execution.
	StrictAutoReview *bool `json:"strictAutoReview"`
}

type GrantedPermissionProfile struct {
	FileSystem *FluffyAdditionalFileSystemPermissions `json:"fileSystem"`
	Network    *FluffyAdditionalNetworkPermissions    `json:"network"`
}

type FluffyAdditionalFileSystemPermissions struct {
	Entries          []FluffyFileSystemSandboxEntry `json:"entries"`
	GlobScanMaxDepth *int64                         `json:"globScanMaxDepth"`
	// This will be removed in favor of `entries`.
	Read []string `json:"read"`
	// This will be removed in favor of `entries`.
	Write []string `json:"write"`
}

type FluffyFileSystemSandboxEntry struct {
	Access FileSystemAccessMode `json:"access"`
	Path   FluffyFileSystemPath `json:"path"`
}

type FluffyFileSystemPath struct {
	Path    *string                      `json:"path,omitempty"`
	Type    FileSystemPathType           `json:"type"`
	Pattern *string                      `json:"pattern,omitempty"`
	Value   *FluffyFileSystemSpecialPath `json:"value,omitempty"`
}

type FluffyFileSystemSpecialPath struct {
	Kind    Kind    `json:"kind"`
	Subpath *string `json:"subpath"`
	Path    *string `json:"path,omitempty"`
}

type FluffyAdditionalNetworkPermissions struct {
	Enabled *bool `json:"enabled"`
}

// Notification sent from the server to the client.
//
// # NEW NOTIFICATIONS
//
// EXPERIMENTAL - proposed plan streaming deltas for plan items.
//
// Stream base64-encoded stdout/stderr chunks for a running `command/exec` session.
//
// Stream base64-encoded stdout/stderr chunks for a running `process/spawn` session.
//
// Final exit notification for a `process/spawn` session.
//
// Deprecated legacy apply_patch output stream notification.
//
// Deprecated: Use `ContextCompaction` item type instead.
//
// Notifies the user of world-writable directories on Windows, which cannot be protected by
// the sandbox.
type ServerNotification struct {
	// Unix timestamp (in milliseconds) when app-server emitted this notification.
	EmittedAtMS *int64             `json:"emittedAtMs,omitempty"`
	Method      NotificationMethod `json:"method"`
	Params      Notification       `json:"params"`
}

// Notification emitted when watched local skill files change.
//
// Treat this as an invalidation signal and re-run `skills/list` with the client's current
// parameters when refreshed skill metadata is needed.
//
// Notification that the turn-level unified diff has changed. Contains the latest aggregated
// diff across all file changes in the turn.
//
// [UNSTABLE] Temporary notification payload for approval auto-review. This shape is
// expected to change soon.
//
// EXPERIMENTAL - proposed plan streaming deltas for plan items. Clients should not assume
// concatenated deltas match the completed plan item content.
//
// Base64-encoded output chunk emitted for a streaming `command/exec` request.
//
// These notifications are connection-scoped. If the originating connection closes, the
// server terminates the process.
//
// Base64-encoded output chunk emitted for a streaming `process/spawn` request.
//
// Final process exit notification for `process/spawn`.
//
// Deprecated legacy notification for `apply_patch` textual output.
//
// The server no longer emits this notification.
//
// Sparse rolling rate-limit update.
//
// Clients should merge available values into the most recent `account/rateLimits/read`
// response or refetch that snapshot. Nullable account metadata may be unavailable in a
// rolling update and does not clear a previously observed value.
//
// EXPERIMENTAL - notification emitted when the app list changes.
//
// Current remote-control connection status and remote identity exposed to clients.
//
// Filesystem watch notification emitted for `fs/watch` subscribers.
//
// Deprecated: Use `ContextCompaction` item type instead.
//
// EXPERIMENTAL - emitted when thread realtime startup is accepted.
//
// EXPERIMENTAL - raw non-audio thread realtime item emitted by the backend.
//
// EXPERIMENTAL - flat transcript delta emitted whenever realtime transcript text changes.
//
// EXPERIMENTAL - final transcript text emitted when realtime completes a transcript part.
//
// EXPERIMENTAL - streamed output audio emitted by thread realtime.
//
// EXPERIMENTAL - emitted with the remote SDP for a WebRTC realtime session.
//
// EXPERIMENTAL - emitted when thread realtime encounters an error.
//
// EXPERIMENTAL - emitted when thread realtime transport closes.
type Notification struct {
	Error *MovePath `json:"error"`
	// Optional thread target when the warning applies to a specific thread.
	//
	// Thread target for the guardian warning.
	ThreadID       *string            `json:"threadId"`
	TurnID         *string            `json:"turnId"`
	WillRetry      *bool              `json:"willRetry,omitempty"`
	Thread         *ThreadClass       `json:"thread,omitempty"`
	Status         *StatusUnion       `json:"status"`
	ThreadName     *string            `json:"threadName"`
	Goal           *ThreadGoal        `json:"goal,omitempty"`
	ChangeType     *ProjectChangeType `json:"changeType,omitempty"`
	ProjectID      *string            `json:"projectId"`
	EnvironmentID  *string            `json:"environmentId"`
	ThreadSettings *ThreadSettings    `json:"threadSettings,omitempty"`
	TokenUsage     *ThreadTokenUsage  `json:"tokenUsage,omitempty"`
	Turn           *Turn              `json:"turn,omitempty"`
	Run            *HookRunSummary    `json:"run,omitempty"`
	Diff           *string            `json:"diff,omitempty"`
	Explanation    *string            `json:"explanation"`
	Plan           []TurnPlanStep     `json:"plan,omitempty"`
	Item           interface{}        `json:"item"`
	// Unix timestamp (in milliseconds) when this item lifecycle started.
	//
	// Unix timestamp (in milliseconds) when this review started.
	StartedAtMS *int64                        `json:"startedAtMs,omitempty"`
	Action      *GuardianApprovalReviewAction `json:"action,omitempty"`
	Review      *GuardianApprovalReview       `json:"review,omitempty"`
	// Stable identifier for this review.
	ReviewID *string `json:"reviewId,omitempty"`
	// Identifier for the reviewed item or tool call when one exists.
	//
	// In most cases, one review maps to one target item. The exceptions are - execve reviews,
	// where a single command may contain multiple execve calls to review (only possible when
	// using the shell_zsh_fork feature) - network policy reviews, where there is no target
	// item
	//
	// A network call is triggered by a CommandExecution item, so having a target_item_id set to
	// the CommandExecution item would be misleading because the review is about the network
	// call, not the command execution. Therefore, target_item_id is set to None for network
	// policy reviews.
	TargetItemID *string `json:"targetItemId"`
	// Unix timestamp (in milliseconds) when this review completed.
	//
	// Unix timestamp (in milliseconds) when this item lifecycle completed.
	CompletedAtMS  *int64                    `json:"completedAtMs,omitempty"`
	DecisionSource *AutoReviewDecisionSource `json:"decisionSource,omitempty"`
	// Live transcript delta from the realtime event.
	Delta  *string `json:"delta,omitempty"`
	ItemID *string `json:"itemId,omitempty"`
	// `true` on the final streamed chunk for a stream when `outputBytesCap` truncated later
	// output on that stream.
	//
	// True on the final streamed chunk for this stream when output was truncated by
	// `outputBytesCap`.
	CapReached *bool `json:"capReached,omitempty"`
	// Base64-encoded output bytes.
	DeltaBase64 *string `json:"deltaBase64,omitempty"`
	// Client-supplied, connection-scoped `processId` from the original `command/exec` request.
	ProcessID *string `json:"processId,omitempty"`
	// Output stream for this chunk.
	//
	// Output stream this chunk belongs to.
	Stream *OutputStream `json:"stream,omitempty"`
	// Client-supplied, connection-scoped `processHandle` from `process/spawn`.
	ProcessHandle *string `json:"processHandle,omitempty"`
	// Process exit code.
	ExitCode *int64 `json:"exitCode,omitempty"`
	// Buffered stderr capture.
	//
	// Empty when stderr was streamed via `process/outputDelta`.
	Stderr *string `json:"stderr,omitempty"`
	// Whether stderr reached `outputBytesCap`.
	//
	// In streaming mode, stderr is empty and cap state is also reported on the final stderr
	// `process/outputDelta` notification.
	StderrCapReached *bool `json:"stderrCapReached,omitempty"`
	// Buffered stdout capture.
	//
	// Empty when stdout was streamed via `process/outputDelta`.
	Stdout *string `json:"stdout,omitempty"`
	// Whether stdout reached `outputBytesCap`.
	//
	// In streaming mode, stdout is empty and cap state is also reported on the final stdout
	// `process/outputDelta` notification.
	StdoutCapReached *bool              `json:"stdoutCapReached,omitempty"`
	Stdin            *string            `json:"stdin,omitempty"`
	Changes          []FileUpdateChange `json:"changes,omitempty"`
	RequestID        *RequestID         `json:"requestId"`
	// Concise warning message for the user.
	//
	// Concise guardian warning message for the user.
	Message         *string                               `json:"message,omitempty"`
	Name            *string                               `json:"name,omitempty"`
	Success         *bool                                 `json:"success,omitempty"`
	FailureReason   *MCPServerStartupFailureReason        `json:"failureReason"`
	AuthMode        *AuthMode                             `json:"authMode"`
	PlanType        *PlanType                             `json:"planType"`
	RateLimits      *RateLimitSnapshot                    `json:"rateLimits,omitempty"`
	Data            []AppInfo                             `json:"data,omitempty"`
	InstallationID  *string                               `json:"installationId,omitempty"`
	ServerName      *string                               `json:"serverName,omitempty"`
	ImportID        *string                               `json:"importId,omitempty"`
	ItemTypeResults []ExternalAgentConfigImportTypeResult `json:"itemTypeResults,omitempty"`
	// File or directory paths associated with this event.
	ChangedPaths []string `json:"changedPaths,omitempty"`
	// Watch identifier previously provided to `fs/watch`.
	WatchID         *string               `json:"watchId,omitempty"`
	SummaryIndex    *int64                `json:"summaryIndex,omitempty"`
	ContentIndex    *int64                `json:"contentIndex,omitempty"`
	FromModel       *string               `json:"fromModel,omitempty"`
	Reason          *string               `json:"reason"`
	ToModel         *string               `json:"toModel,omitempty"`
	Verifications   []VerificationElement `json:"verifications,omitempty"`
	Metadata        interface{}           `json:"metadata"`
	FasterModel     *string               `json:"fasterModel"`
	Model           *string               `json:"model,omitempty"`
	Reasons         []string              `json:"reasons,omitempty"`
	ShowBufferingUI *bool                 `json:"showBufferingUi,omitempty"`
	UseCases        []string              `json:"useCases,omitempty"`
	// Optional extra guidance, such as migration steps or rationale.
	//
	// Optional extra guidance or error details.
	Details *string `json:"details"`
	// Concise summary of what is deprecated.
	//
	// Concise summary of the warning.
	Summary *string `json:"summary,omitempty"`
	// Optional path to the config file that triggered the warning.
	Path *string `json:"path"`
	// Optional range for the error location inside the config file.
	Range             *TextRange                   `json:"range"`
	Files             []ParamsFile                 `json:"files,omitempty"`
	Query             *string                      `json:"query,omitempty"`
	SessionID         *string                      `json:"sessionId,omitempty"`
	RealtimeSessionID *string                      `json:"realtimeSessionId"`
	Version           *RealtimeConversationVersion `json:"version,omitempty"`
	Role              *string                      `json:"role,omitempty"`
	// Final complete text for the transcript part.
	Text                 *string                      `json:"text,omitempty"`
	Audio                *ThreadRealtimeAudioChunk    `json:"audio,omitempty"`
	SDP                  *string                      `json:"sdp,omitempty"`
	ExtraCount           *int64                       `json:"extraCount,omitempty"`
	FailedScan           *bool                        `json:"failedScan,omitempty"`
	SamplePaths          []string                     `json:"samplePaths,omitempty"`
	Mode                 *WindowsSandboxSetupMode     `json:"mode,omitempty"`
	LoginID              *string                      `json:"loginId"`
	OnboardingEntrypoint *DesktopOnboardingEntrypoint `json:"onboardingEntrypoint"`
}

type GuardianApprovalReviewAction struct {
	Command       *string                                  `json:"command,omitempty"`
	Cwd           *string                                  `json:"cwd,omitempty"`
	Source        *GuardianCommandSource                   `json:"source,omitempty"`
	Type          GuardianApprovalReviewActionType         `json:"type"`
	Argv          []string                                 `json:"argv,omitempty"`
	Program       *string                                  `json:"program,omitempty"`
	Files         []string                                 `json:"files,omitempty"`
	Host          *string                                  `json:"host,omitempty"`
	Port          *int64                                   `json:"port,omitempty"`
	Protocol      *NetworkApprovalProtocol                 `json:"protocol,omitempty"`
	Target        *string                                  `json:"target,omitempty"`
	ConnectorID   *string                                  `json:"connectorId"`
	ConnectorName *string                                  `json:"connectorName"`
	Server        *string                                  `json:"server,omitempty"`
	ToolName      *string                                  `json:"toolName,omitempty"`
	ToolTitle     *string                                  `json:"toolTitle"`
	Permissions   *GuardianApprovalReviewActionPermissions `json:"permissions,omitempty"`
	Reason        *string                                  `json:"reason"`
}

type GuardianApprovalReviewActionPermissions struct {
	FileSystem *TentacledAdditionalFileSystemPermissions `json:"fileSystem"`
	Network    *TentacledAdditionalNetworkPermissions    `json:"network"`
}

type TentacledAdditionalFileSystemPermissions struct {
	Entries          []TentacledFileSystemSandboxEntry `json:"entries"`
	GlobScanMaxDepth *int64                            `json:"globScanMaxDepth"`
	// This will be removed in favor of `entries`.
	Read []string `json:"read"`
	// This will be removed in favor of `entries`.
	Write []string `json:"write"`
}

type TentacledFileSystemSandboxEntry struct {
	Access FileSystemAccessMode    `json:"access"`
	Path   TentacledFileSystemPath `json:"path"`
}

type TentacledFileSystemPath struct {
	Path    *string                         `json:"path,omitempty"`
	Type    FileSystemPathType              `json:"type"`
	Pattern *string                         `json:"pattern,omitempty"`
	Value   *TentacledFileSystemSpecialPath `json:"value,omitempty"`
}

type TentacledFileSystemSpecialPath struct {
	Kind    Kind    `json:"kind"`
	Subpath *string `json:"subpath"`
	Path    *string `json:"path,omitempty"`
}

type TentacledAdditionalNetworkPermissions struct {
	Enabled *bool `json:"enabled"`
}

// EXPERIMENTAL - thread realtime audio chunk.
type ThreadRealtimeAudioChunk struct {
	Data              string  `json:"data"`
	ItemID            *string `json:"itemId"`
	NumChannels       int64   `json:"numChannels"`
	SampleRate        int64   `json:"sampleRate"`
	SamplesPerChannel *int64  `json:"samplesPerChannel"`
}

type FileUpdateChange struct {
	Diff string          `json:"diff"`
	Kind PatchChangeKind `json:"kind"`
	Path string          `json:"path"`
}

type PatchChangeKind struct {
	Type     Type    `json:"type"`
	MovePath *string `json:"move_path"`
}

// EXPERIMENTAL - app metadata returned by app-list APIs.
type AppInfo struct {
	AppMetadata         *AppMetadata      `json:"appMetadata"`
	Branding            *AppBranding      `json:"branding"`
	Description         *string           `json:"description"`
	DistributionChannel *string           `json:"distributionChannel"`
	IconAssets          map[string]string `json:"iconAssets"`
	IconDarkAssets      map[string]string `json:"iconDarkAssets"`
	ID                  string            `json:"id"`
	InstallURL          *string           `json:"installUrl"`
	IsAccessible        *bool             `json:"isAccessible,omitempty"`
	// Whether this app is enabled in config.toml. Example: ```toml [apps.bad_app] enabled =
	// false ```
	IsEnabled          *bool             `json:"isEnabled,omitempty"`
	Labels             map[string]string `json:"labels"`
	LogoURL            *string           `json:"logoUrl"`
	LogoURLDark        *string           `json:"logoUrlDark"`
	Name               string            `json:"name"`
	PluginDisplayNames []string          `json:"pluginDisplayNames,omitempty"`
}

type AppMetadata struct {
	Categories                 []string        `json:"categories"`
	Developer                  *string         `json:"developer"`
	FirstPartyRequiresInstall  *bool           `json:"firstPartyRequiresInstall"`
	Review                     *AppReview      `json:"review"`
	Screenshots                []AppScreenshot `json:"screenshots"`
	SEODescription             *string         `json:"seoDescription"`
	ShowInComposerWhenUnlinked *bool           `json:"showInComposerWhenUnlinked"`
	SubCategories              []string        `json:"subCategories"`
	Version                    *string         `json:"version"`
	VersionID                  *string         `json:"versionId"`
	VersionNotes               *string         `json:"versionNotes"`
}

type AppReview struct {
	Status string `json:"status"`
}

type AppScreenshot struct {
	FileID     *string `json:"fileId"`
	URL        *string `json:"url"`
	UserPrompt string  `json:"userPrompt"`
}

// EXPERIMENTAL - app metadata returned by app-list APIs.
type AppBranding struct {
	Category          *string `json:"category"`
	Developer         *string `json:"developer"`
	IsDiscoverableApp bool    `json:"isDiscoverableApp"`
	PrivacyPolicy     *string `json:"privacyPolicy"`
	TermsOfService    *string `json:"termsOfService"`
	Website           *string `json:"website"`
}

type TurnError struct {
	AdditionalDetails *string              `json:"additionalDetails"`
	CodexErrorInfo    *CodexErrorInfoUnion `json:"codexErrorInfo"`
	Message           string               `json:"message"`
}

// Failed to connect to the response SSE stream.
//
// The response SSE stream disconnected in the middle of a turn before completion.
//
// Reached the retry limit for responses.
//
// Returned when `turn/start` or `turn/steer` is submitted while the current active turn
// cannot accept same-turn steering, for example `/review` or manual `/compact`.
type CodexErrorInfo struct {
	HTTPConnectionFailed           *HTTPConnectionFailed           `json:"httpConnectionFailed,omitempty"`
	ResponseStreamConnectionFailed *ResponseStreamConnectionFailed `json:"responseStreamConnectionFailed,omitempty"`
	ResponseStreamDisconnected     *ResponseStreamDisconnected     `json:"responseStreamDisconnected,omitempty"`
	ResponseTooManyFailedAttempts  *ResponseTooManyFailedAttempts  `json:"responseTooManyFailedAttempts,omitempty"`
	ActiveTurnNotSteerable         *ActiveTurnNotSteerable         `json:"activeTurnNotSteerable,omitempty"`
}

type ActiveTurnNotSteerable struct {
	TurnKind NonSteerableTurnKind `json:"turnKind"`
}

type HTTPConnectionFailed struct {
	HTTPStatusCode *int64 `json:"httpStatusCode"`
}

type ResponseStreamConnectionFailed struct {
	HTTPStatusCode *int64 `json:"httpStatusCode"`
}

type ResponseStreamDisconnected struct {
	HTTPStatusCode *int64 `json:"httpStatusCode"`
}

type ResponseTooManyFailedAttempts struct {
	HTTPStatusCode *int64 `json:"httpStatusCode"`
}

// Superset of [`codex_file_search::FileMatch`]
type ParamsFile struct {
	FileName  string                   `json:"file_name"`
	Indices   []int64                  `json:"indices"`
	MatchType FuzzyFileSearchMatchType `json:"match_type"`
	Path      string                   `json:"path"`
	Root      string                   `json:"root"`
	Score     int64                    `json:"score"`
}

type ThreadGoal struct {
	CreatedAt       int64            `json:"createdAt"`
	Objective       string           `json:"objective"`
	Status          ThreadGoalStatus `json:"status"`
	ThreadID        string           `json:"threadId"`
	TimeUsedSeconds int64            `json:"timeUsedSeconds"`
	TokenBudget     *int64           `json:"tokenBudget"`
	TokensUsed      int64            `json:"tokensUsed"`
	UpdatedAt       int64            `json:"updatedAt"`
}

type ExternalAgentConfigImportTypeResult struct {
	Failures  []FluffyExternalAgentConfigImportItemTypeFailure `json:"failures"`
	ItemType  ExternalAgentConfigMigrationItemType             `json:"itemType"`
	Successes []ExternalAgentConfigImportItemTypeSuccess       `json:"successes"`
}

type FluffyExternalAgentConfigImportItemTypeFailure struct {
	Cwd          *string                              `json:"cwd"`
	ErrorType    *string                              `json:"errorType"`
	FailureStage string                               `json:"failureStage"`
	ItemType     ExternalAgentConfigMigrationItemType `json:"itemType"`
	Message      string                               `json:"message"`
	Source       *string                              `json:"source"`
	SubErrorType *string                              `json:"subErrorType"`
}

type ExternalAgentConfigImportItemTypeSuccess struct {
	Cwd      *string                              `json:"cwd"`
	ItemType ExternalAgentConfigMigrationItemType `json:"itemType"`
	Source   *string                              `json:"source"`
	Target   *string                              `json:"target"`
	// Original title for an imported session; null for other item types.
	Title *string `json:"title"`
}

type TurnPlanStep struct {
	Status TurnPlanStepStatus `json:"status"`
	Step   string             `json:"step"`
}

type TextRange struct {
	End   TextPosition `json:"end"`
	Start TextPosition `json:"start"`
}

type TextPosition struct {
	// 1-based column number (in Unicode scalar values).
	Column int64 `json:"column"`
	// 1-based line number.
	Line int64 `json:"line"`
}

type RateLimitSnapshot struct {
	Credits              *CreditsSnapshot           `json:"credits"`
	IndividualLimit      *SpendControlLimitSnapshot `json:"individualLimit"`
	LimitID              *string                    `json:"limitId"`
	LimitName            *string                    `json:"limitName"`
	PlanType             *PlanType                  `json:"planType"`
	Primary              *RateLimitWindow           `json:"primary"`
	RateLimitReachedType *RateLimitReachedType      `json:"rateLimitReachedType"`
	Secondary            *RateLimitWindow           `json:"secondary"`
	// Backend-reported spend-control state. `None` is unavailable, not a sparse-update recovery.
	SpendControlReached *bool `json:"spendControlReached"`
}

type CreditsSnapshot struct {
	Balance    *string `json:"balance"`
	HasCredits bool    `json:"hasCredits"`
	Unlimited  bool    `json:"unlimited"`
}

type SpendControlLimitSnapshot struct {
	Limit            string `json:"limit"`
	RemainingPercent int64  `json:"remainingPercent"`
	ResetsAt         int64  `json:"resetsAt"`
	Used             string `json:"used"`
}

type RateLimitWindow struct {
	ResetsAt           *int64 `json:"resetsAt"`
	UsedPercent        int64  `json:"usedPercent"`
	WindowDurationMins *int64 `json:"windowDurationMins"`
}

// [UNSTABLE] Temporary approval auto-review payload used by `item/autoApprovalReview/*`
// notifications. This shape is expected to change soon.
type GuardianApprovalReview struct {
	Rationale         *string                      `json:"rationale"`
	RiskLevel         *GuardianRiskLevel           `json:"riskLevel"`
	Status            GuardianApprovalReviewStatus `json:"status"`
	UserAuthorization *GuardianUserAuthorization   `json:"userAuthorization"`
}

type HookRunSummary struct {
	CompletedAt   *int64            `json:"completedAt"`
	DisplayOrder  int64             `json:"displayOrder"`
	DurationMS    *int64            `json:"durationMs"`
	Entries       []HookOutputEntry `json:"entries"`
	EventName     HookEventName     `json:"eventName"`
	ExecutionMode HookExecutionMode `json:"executionMode"`
	HandlerType   HookHandlerType   `json:"handlerType"`
	ID            string            `json:"id"`
	Scope         HookScope         `json:"scope"`
	Source        *HookSource       `json:"source,omitempty"`
	SourcePath    string            `json:"sourcePath"`
	StartedAt     int64             `json:"startedAt"`
	Status        HookRunStatus     `json:"status"`
	StatusMessage *string           `json:"statusMessage"`
}

type HookOutputEntry struct {
	Kind HookOutputEntryKind `json:"kind"`
	Text string              `json:"text"`
}

type ThreadStatus struct {
	Type        ThreadStatusType   `json:"type"`
	ActiveFlags []ThreadActiveFlag `json:"activeFlags,omitempty"`
}

type ThreadClass struct {
	// Optional random unique nickname assigned to an AgentControl-spawned sub-agent.
	AgentNickname *string `json:"agentNickname"`
	// Optional role (agent_role) assigned to an AgentControl-spawned sub-agent.
	AgentRole *string `json:"agentRole"`
	// Version of the CLI that created the thread.
	CLIVersion string `json:"cliVersion"`
	// Unix timestamp (in seconds) when the thread was created.
	CreatedAt int64 `json:"createdAt"`
	// Working directory captured for the thread.
	Cwd string `json:"cwd"`
	// Whether the thread is ephemeral and should not be materialized on disk.
	Ephemeral bool `json:"ephemeral"`
	// Source thread id when this thread was created by forking another thread.
	ForkedFromID *string `json:"forkedFromId"`
	// Optional Git metadata captured when the thread was created.
	GitInfo *GitInfo `json:"gitInfo"`
	// Identifier for this thread. Codex-generated thread IDs are UUIDv7.
	ID string `json:"id"`
	// Model provider used for this thread (for example, 'openai').
	ModelProvider string `json:"modelProvider"`
	// Optional user-facing thread title.
	Name *string `json:"name"`
	// The ID of the parent thread. This will only be set if this thread is a subagent.
	ParentThreadID *string `json:"parentThreadId"`
	// [UNSTABLE] Path to the thread on disk.
	Path *string `json:"path"`
	// Usually the first user message in the thread, if available.
	Preview string `json:"preview"`
	// Canonical project assignment owned by app-server, if any.
	ProjectID *string `json:"projectId"`
	// Unix timestamp (in seconds) used for thread recency ordering.
	RecencyAt *int64 `json:"recencyAt"`
	// The independently persisted section selected for this thread, if any.
	Section *ThreadSection `json:"section"`
	// Unix timestamp in seconds when the thread entered its current section.
	SectionEnteredAt *int64 `json:"sectionEnteredAt"`
	// Session id shared by threads that belong to the same session tree.
	SessionID string `json:"sessionId"`
	// Origin of the thread (CLI, VSCode, codex exec, codex app-server, etc.).
	Source *SessionSourceUnion `json:"source"`
	// Current runtime status for the thread.
	Status ThreadStatusClass `json:"status"`
	// Optional analytics source classification for this thread.
	ThreadSource *string `json:"threadSource"`
	// Only populated on `thread/resume`, `thread/rollback`, `thread/fork`, and `thread/read`
	// (when `includeTurns` is true) responses. For all other responses and notifications
	// returning a Thread, the turns field will be an empty list.
	Turns []Turn `json:"turns"`
	// Unix timestamp (in seconds) when the thread was last updated.
	UpdatedAt int64 `json:"updatedAt"`
}

type GitInfo struct {
	Branch    *string `json:"branch"`
	OriginURL *string `json:"originUrl"`
	SHA       *string `json:"sha"`
}

// An independently persisted, user-visible thread section.
type ThreadSection struct {
	// Optional appearance synchronized across clients.
	Appearance *ThreadSectionThreadSectionAppearance `json:"appearance"`
	// Opaque UUIDv7 identity that remains stable when the section is renamed.
	ID string `json:"id"`
	// The current user-visible section name.
	Name string `json:"name"`
}

// Extensible visual presentation for a custom thread section.
type ThreadSectionThreadSectionAppearance struct {
	Color *string `json:"color"`
	Icon  *string `json:"icon"`
}

type SessionSource struct {
	Custom   *string              `json:"custom,omitempty"`
	SubAgent *SubAgentSourceUnion `json:"subAgent"`
}

type SubAgentSource struct {
	ThreadSpawn *ThreadSpawn `json:"thread_spawn,omitempty"`
	Other       *string      `json:"other,omitempty"`
}

type ThreadSpawn struct {
	AgentNickname  *string `json:"agent_nickname"`
	AgentPath      *string `json:"agent_path"`
	AgentRole      *string `json:"agent_role"`
	Depth          int64   `json:"depth"`
	ParentThreadID string  `json:"parent_thread_id"`
}

// Current runtime status for the thread.
type ThreadStatusClass struct {
	Type        ThreadStatusType   `json:"type"`
	ActiveFlags []ThreadActiveFlag `json:"activeFlags,omitempty"`
}

type Turn struct {
	// Unix timestamp (in seconds) when the turn completed.
	CompletedAt *int64 `json:"completedAt"`
	// Duration between turn start and completion in milliseconds, if known.
	DurationMS *int64 `json:"durationMs"`
	// Only populated when the Turn's status is failed.
	Error *TurnError `json:"error"`
	// Identifier for this turn. Codex-generated turn IDs are UUIDv7.
	ID string `json:"id"`
	// Thread items currently included in this turn payload.
	Items []ThreadItem `json:"items"`
	// Describes how much of `items` has been loaded for this turn.
	ItemsView *TurnItemsView `json:"itemsView,omitempty"`
	// Unix timestamp (in seconds) when the turn started.
	StartedAt *int64     `json:"startedAt"`
	Status    TurnStatus `json:"status"`
}

// EXPERIMENTAL - proposed plan item content. The completed plan item is authoritative and
// may not match the concatenation of `PlanDelta` text.
//
// Display item emitted by the interruptible `clock.sleep` tool.
type ThreadItem struct {
	ClientID *string          `json:"clientId"`
	Content  []ContentElement `json:"content,omitempty"`
	// Unique identifier for this collab tool call.
	ID             string                `json:"id"`
	Type           ThreadItemType        `json:"type"`
	Fragments      []HookPromptFragment  `json:"fragments,omitempty"`
	Delivery       *AgentMessageDelivery `json:"delivery"`
	MemoryCitation *MemoryCitation       `json:"memoryCitation"`
	Phase          *MessagePhase         `json:"phase"`
	Text           *string               `json:"text,omitempty"`
	Summary        []string              `json:"summary,omitempty"`
	// The command's output, aggregated from stdout and stderr.
	AggregatedOutput *string `json:"aggregatedOutput"`
	// The command to be executed.
	Command *string `json:"command,omitempty"`
	// A best-effort parsing of the command to understand the action(s) it will perform. This
	// returns a list of CommandAction objects because a single shell command may be composed of
	// many commands piped together.
	CommandActions []ThreadItemCommandAction `json:"commandActions,omitempty"`
	// The command's working directory.
	Cwd *string `json:"cwd,omitempty"`
	// The duration of the command execution in milliseconds.
	//
	// The duration of the MCP tool call in milliseconds.
	//
	// The duration of the dynamic tool call in milliseconds.
	DurationMS *int64 `json:"durationMs"`
	// The command's exit code.
	ExitCode *int64 `json:"exitCode"`
	// Trusted first-party plugin id when this command resolves to one plugin script.
	PluginID *string `json:"pluginId"`
	// Identifier for the underlying PTY process (when available).
	ProcessID *string `json:"processId"`
	// Safe plugin-relative path when this command resolves to one plugin script.
	ScriptPath *string                 `json:"scriptPath"`
	Source     *CommandExecutionSource `json:"source,omitempty"`
	// Current status of the collab tool call.
	Status     *string                `json:"status,omitempty"`
	Changes    []FileUpdateChange     `json:"changes,omitempty"`
	AppContext *MCPToolCallAppContext `json:"appContext"`
	Arguments  interface{}            `json:"arguments"`
	Error      *MCPToolCallError      `json:"error"`
	// Deprecated: use `appContext.resourceUri` instead.
	MCPAppResourceURI *string `json:"mcpAppResourceUri"`
	ReadOnlyHint      *bool   `json:"readOnlyHint"`
	Result            *Result `json:"result"`
	Server            *string `json:"server,omitempty"`
	// Name of the collab tool that was invoked.
	Tool         *string                                      `json:"tool,omitempty"`
	ContentItems []ThreadItemDynamicToolCallOutputContentItem `json:"contentItems"`
	Namespace    *string                                      `json:"namespace"`
	Success      *bool                                        `json:"success"`
	// Last known status of the target agents, when available.
	AgentsStates map[string]CollabAgentState `json:"agentsStates,omitempty"`
	// Model requested for the spawned agent, when applicable.
	Model *string `json:"model"`
	// Prompt text sent as part of the collab tool call, when available.
	Prompt *string `json:"prompt"`
	// Reasoning effort requested for the spawned agent, when applicable.
	ReasoningEffort *string `json:"reasoningEffort"`
	// Thread ID of the receiving agent, when applicable. In case of spawn operation, this
	// corresponds to the newly spawned agent.
	ReceiverThreadIDS []string `json:"receiverThreadIds,omitempty"`
	// Thread ID of the agent issuing the collab request.
	SenderThreadID *string               `json:"senderThreadId,omitempty"`
	AgentPath      *string               `json:"agentPath,omitempty"`
	AgentThreadID  *string               `json:"agentThreadId,omitempty"`
	Kind           *SubAgentActivityKind `json:"kind,omitempty"`
	Action         *WebSearchAction      `json:"action"`
	Query          *string               `json:"query,omitempty"`
	// Structured search results returned out-of-band by standalone web search.
	//
	// These stay as opaque JSON at the extension/app-server boundary so new result fields and
	// result types can pass through without a Codex release.
	Results               []interface{}           `json:"results"`
	Path                  *string                 `json:"path,omitempty"`
	Failure               *ImageGenerationFailure `json:"failure"`
	RevisedPrompt         *string                 `json:"revisedPrompt"`
	SavedPath             *string                 `json:"savedPath"`
	TransparentBackground *bool                   `json:"transparentBackground"`
	Review                *string                 `json:"review,omitempty"`
}

type WebSearchAction struct {
	Queries []string            `json:"queries"`
	Query   *string             `json:"query"`
	Type    WebSearchActionType `json:"type"`
	URL     *string             `json:"url"`
	Pattern *string             `json:"pattern"`
}

type CollabAgentState struct {
	Message *string           `json:"message"`
	Status  CollabAgentStatus `json:"status"`
}

type MCPToolCallAppContext struct {
	ActionName  *string `json:"actionName"`
	AppName     *string `json:"appName"`
	ConnectorID string  `json:"connectorId"`
	LinkID      *string `json:"linkId"`
	ResourceURI *string `json:"resourceUri"`
}

type ThreadItemCommandAction struct {
	Command string            `json:"command"`
	Name    *string           `json:"name,omitempty"`
	Path    *string           `json:"path"`
	Type    CommandActionType `json:"type"`
	Query   *string           `json:"query"`
}

type TextUserInputClass struct {
	Text *string `json:"text,omitempty"`
	// UI-defined spans within `text` used to render or persist special elements.
	TextElements []TextUserInputTextElement `json:"text_elements,omitempty"`
	Type         UserInputType              `json:"type"`
	Detail       *ImageDetail               `json:"detail"`
	URL          *string                    `json:"url,omitempty"`
	Path         *string                    `json:"path,omitempty"`
	Name         *string                    `json:"name,omitempty"`
}

type TextUserInputTextElement struct {
	// Byte range in the parent `text` buffer that this element occupies.
	ByteRange FluffyByteRange `json:"byteRange"`
	// Optional human-readable placeholder for the element, displayed in the UI.
	Placeholder *string `json:"placeholder"`
}

// Byte range in the parent `text` buffer that this element occupies.
type FluffyByteRange struct {
	End   int64 `json:"end"`
	Start int64 `json:"start"`
}

type ThreadItemDynamicToolCallOutputContentItem struct {
	Text     *string                                   `json:"text,omitempty"`
	Type     InputDynamicToolCallOutputContentItemType `json:"type"`
	ImageURL *string                                   `json:"imageUrl,omitempty"`
	AudioURL *string                                   `json:"audioUrl,omitempty"`
}

type MCPToolCallError struct {
	Message string `json:"message"`
}

type ImageGenerationFailure struct {
	LimitID  string                                       `json:"limitId"`
	ResetsAt *int64                                       `json:"resetsAt"`
	Type     UsageLimitExceededImageGenerationFailureType `json:"type"`
}

type HookPromptFragment struct {
	HookRunID string `json:"hookRunId"`
	Text      string `json:"text"`
}

type MemoryCitation struct {
	Entries   []MemoryCitationEntry `json:"entries"`
	ThreadIDS []string              `json:"threadIds"`
}

type MemoryCitationEntry struct {
	LineEnd   int64  `json:"lineEnd"`
	LineStart int64  `json:"lineStart"`
	Note      string `json:"note"`
	Path      string `json:"path"`
}

type MCPToolCallResult struct {
	Meta              interface{}   `json:"_meta"`
	Content           []interface{} `json:"content"`
	StructuredContent interface{}   `json:"structuredContent"`
}

type ThreadSettings struct {
	ActivePermissionProfile *ActivePermissionProfile `json:"activePermissionProfile"`
	ApprovalPolicy          *AskForApproval          `json:"approvalPolicy"`
	ApprovalsReviewer       ApprovalsReviewer        `json:"approvalsReviewer"`
	CollaborationMode       CollaborationMode        `json:"collaborationMode"`
	Cwd                     string                   `json:"cwd"`
	Effort                  *string                  `json:"effort"`
	Model                   string                   `json:"model"`
	ModelProvider           string                   `json:"modelProvider"`
	Personality             *Personality             `json:"personality"`
	SandboxPolicy           SandboxPolicyClass       `json:"sandboxPolicy"`
	ServiceTier             *string                  `json:"serviceTier"`
	Summary                 *ReasoningSummary        `json:"summary"`
}

type ActivePermissionProfile struct {
	// Parent profile identifier from the selected permissions profile's `extends` setting, when
	// present.
	Extends *string `json:"extends"`
	// Identifier from `default_permissions` or the implicit built-in default, such as
	// `:workspace` or a user-defined `[permissions.<id>]` profile.
	ID string `json:"id"`
}

type AskForApprovalGranularAskForApproval struct {
	Granular FluffyGranular `json:"granular"`
}

type FluffyGranular struct {
	MCPElicitations    bool  `json:"mcp_elicitations"`
	RequestPermissions *bool `json:"request_permissions,omitempty"`
	Rules              bool  `json:"rules"`
	SandboxApproval    bool  `json:"sandbox_approval"`
	SkillApproval      *bool `json:"skill_approval,omitempty"`
}

// Collaboration mode for a Codex session.
type CollaborationMode struct {
	Mode     ModeKind `json:"mode"`
	Settings Settings `json:"settings"`
}

// Settings for a collaboration mode.
type Settings struct {
	DeveloperInstructions *string `json:"developer_instructions"`
	Model                 string  `json:"model"`
	ReasoningEffort       *string `json:"reasoning_effort"`
}

type SandboxPolicyClass struct {
	Type                SandboxPolicyType   `json:"type"`
	NetworkAccess       *NetworkAccessUnion `json:"networkAccess"`
	ExcludeSlashTmp     *bool               `json:"excludeSlashTmp,omitempty"`
	ExcludeTmpdirEnvVar *bool               `json:"excludeTmpdirEnvVar,omitempty"`
	WritableRoots       []string            `json:"writableRoots,omitempty"`
}

type ThreadTokenUsage struct {
	Last               TokenUsageBreakdown `json:"last"`
	ModelContextWindow *int64              `json:"modelContextWindow"`
	Total              TokenUsageBreakdown `json:"total"`
}

type TokenUsageBreakdown struct {
	CachedInputTokens     int64  `json:"cachedInputTokens"`
	CacheWriteInputTokens *int64 `json:"cacheWriteInputTokens,omitempty"`
	InputTokens           int64  `json:"inputTokens"`
	OutputTokens          int64  `json:"outputTokens"`
	ReasoningOutputTokens int64  `json:"reasoningOutputTokens"`
	TotalTokens           int64  `json:"totalTokens"`
}

// Request initiated from the server and sent to the client.
//
// NEW APIs Sent when approval is requested for a specific command execution. This request
// is used for Turns started via turn/start.
//
// Sent when approval is requested for a specific file change. This request is used for
// Turns started via turn/start.
//
// EXPERIMENTAL - Request input from the user for a tool call.
//
// Request input for an MCP server elicitation.
//
// Request approval for additional permissions from the user.
//
// Execute a dynamic tool call on the client.
//
// Generate a fresh upstream attestation result on demand.
//
// DEPRECATED APIs below Request to approve a patch. This request is used for Turns started
// via the legacy APIs (i.e. SendUserTurn, SendUserMessage).
//
// Request to exec a command. This request is used for Turns started via the legacy APIs
// (i.e. SendUserTurn, SendUserMessage).
type ServerRequest struct {
	ID     *RequestID          `json:"id"`
	Method ServerRequestMethod `json:"method"`
	Params ParamsClass         `json:"params"`
}

// EXPERIMENTAL. Params sent with a request_user_input event.
type ParamsClass struct {
	// Unique identifier for this specific approval callback.
	//
	// For regular shell/unified_exec approvals, this is null.
	//
	// For zsh-exec-bridge subcommand approvals, multiple callbacks can belong to one parent
	// `itemId`, so `approvalId` is a distinct opaque callback id (a UUID) used to disambiguate
	// routing.
	//
	// Identifier for this specific approval callback.
	ApprovalID *string `json:"approvalId"`
	// The command to be executed.
	Command *ThreadListCwdFilter `json:"command"`
	// Best-effort parsed command actions for friendly display.
	CommandActions []ParamsCommandAction `json:"commandActions"`
	// The command's working directory.
	Cwd *string `json:"cwd"`
	// Environment in which the command will run.
	EnvironmentID *string `json:"environmentId"`
	ItemID        *string `json:"itemId,omitempty"`
	// Optional context for a managed-network approval prompt.
	NetworkApprovalContext *ParamsNetworkApprovalContext `json:"networkApprovalContext"`
	// Optional proposed execpolicy amendment to allow similar commands without prompting.
	ProposedExecpolicyAmendment []string `json:"proposedExecpolicyAmendment"`
	// Optional proposed network policy amendments (allow/deny host) for future requests.
	ProposedNetworkPolicyAmendments []ParamsProposedNetworkPolicyAmendment `json:"proposedNetworkPolicyAmendments"`
	// Optional explanatory reason (e.g. request for network access).
	//
	// Optional explanatory reason (e.g. request for extra write access).
	Reason *string `json:"reason"`
	// Unix timestamp (in milliseconds) when this approval request started.
	StartedAtMS *int64  `json:"startedAtMs,omitempty"`
	ThreadID    *string `json:"threadId,omitempty"`
	// Active Codex turn when this elicitation was observed, if app-server could correlate one.
	//
	// This is nullable because MCP models elicitation as a standalone server-to-client request
	// identified by the MCP server request id. It may be triggered during a turn, but turn
	// context is app-server correlation rather than part of the protocol identity of the
	// elicitation itself.
	TurnID *string `json:"turnId"`
	// [UNSTABLE] When set, the agent is asking the user to allow writes under this root for the
	// remainder of the session (unclear if this is honored today).
	//
	// When set, the agent is asking the user to allow writes under this root for the remainder
	// of the session (unclear if this is honored today).
	GrantRoot *string `json:"grantRoot"`
	// @deprecated Use `isBlocking` to decide whether the request should block.
	AutoResolutionMS *int64             `json:"autoResolutionMs"`
	IsBlocking       *bool              `json:"isBlocking,omitempty"`
	Questions        []ParamsQuestion   `json:"questions,omitempty"`
	ServerName       *string            `json:"serverName,omitempty"`
	Meta             interface{}        `json:"_meta"`
	Message          *string            `json:"message,omitempty"`
	Mode             *Mode              `json:"mode,omitempty"`
	RequestedSchema  interface{}        `json:"requestedSchema"`
	ElicitationID    *string            `json:"elicitationId,omitempty"`
	URL              *string            `json:"url,omitempty"`
	Permissions      *ParamsPermissions `json:"permissions,omitempty"`
	Arguments        interface{}        `json:"arguments"`
	// Use to correlate this with [codex_protocol::protocol::PatchApplyBeginEvent] and
	// [codex_protocol::protocol::PatchApplyEndEvent].
	//
	// Use to correlate this with [codex_protocol::protocol::ExecCommandBeginEvent] and
	// [codex_protocol::protocol::ExecCommandEndEvent].
	CallID    *string `json:"callId,omitempty"`
	Namespace *string `json:"namespace"`
	Tool      *string `json:"tool,omitempty"`
	// Workspace/account identifier that Codex was previously using.
	//
	// Clients that manage multiple accounts/workspaces can use this as a hint to refresh the
	// token for the correct workspace.
	//
	// This may be `null` when the prior auth state did not include a workspace identifier
	// (`chatgpt_account_id`).
	PreviousAccountID *string                     `json:"previousAccountId"`
	ConversationID    *string                     `json:"conversationId,omitempty"`
	FileChanges       map[string]ParamsFileChange `json:"fileChanges,omitempty"`
	ParsedCmd         []ParamsParsedCommand       `json:"parsedCmd,omitempty"`
}

type ParamsCommandAction struct {
	Command string            `json:"command"`
	Name    *string           `json:"name,omitempty"`
	Path    *string           `json:"path"`
	Type    CommandActionType `json:"type"`
	Query   *string           `json:"query"`
}

type ParamsFileChange struct {
	Content     *string `json:"content,omitempty"`
	Type        Type    `json:"type"`
	MovePath    *string `json:"move_path"`
	UnifiedDiff *string `json:"unified_diff,omitempty"`
}

type ParamsNetworkApprovalContext struct {
	Host     string                  `json:"host"`
	Protocol NetworkApprovalProtocol `json:"protocol"`
}

type ParamsParsedCommand struct {
	Cmd  string  `json:"cmd"`
	Name *string `json:"name,omitempty"`
	// (Best effort) Path to the file being read by the command. When possible, this is an
	// absolute path, though when relative, it should be resolved against the `cwd`` that will
	// be used to run the command to derive the absolute path.
	Path  *string           `json:"path"`
	Type  ParsedCommandType `json:"type"`
	Query *string           `json:"query"`
}

type ParamsPermissions struct {
	FileSystem *StickyAdditionalFileSystemPermissions `json:"fileSystem"`
	Network    *StickyAdditionalNetworkPermissions    `json:"network"`
}

type StickyAdditionalFileSystemPermissions struct {
	Entries          []StickyFileSystemSandboxEntry `json:"entries"`
	GlobScanMaxDepth *int64                         `json:"globScanMaxDepth"`
	// This will be removed in favor of `entries`.
	Read []string `json:"read"`
	// This will be removed in favor of `entries`.
	Write []string `json:"write"`
}

type StickyFileSystemSandboxEntry struct {
	Access FileSystemAccessMode `json:"access"`
	Path   StickyFileSystemPath `json:"path"`
}

type StickyFileSystemPath struct {
	Path    *string                      `json:"path,omitempty"`
	Type    FileSystemPathType           `json:"type"`
	Pattern *string                      `json:"pattern,omitempty"`
	Value   *StickyFileSystemSpecialPath `json:"value,omitempty"`
}

type StickyFileSystemSpecialPath struct {
	Kind    Kind    `json:"kind"`
	Subpath *string `json:"subpath"`
	Path    *string `json:"path,omitempty"`
}

type StickyAdditionalNetworkPermissions struct {
	Enabled *bool `json:"enabled"`
}

type ParamsProposedNetworkPolicyAmendment struct {
	Action NetworkPolicyRuleAction `json:"action"`
	Host   string                  `json:"host"`
}

// EXPERIMENTAL. Represents one request_user_input question and its required options.
type ParamsQuestion struct {
	Header   string                             `json:"header"`
	ID       string                             `json:"id"`
	IsOther  *bool                              `json:"isOther,omitempty"`
	IsSecret *bool                              `json:"isSecret,omitempty"`
	Options  []PurpleToolRequestUserInputOption `json:"options"`
	Question string                             `json:"question"`
}

// EXPERIMENTAL. Defines a single selectable option for request_user_input.
type PurpleToolRequestUserInputOption struct {
	Description string `json:"description"`
	Label       string `json:"label"`
}

// EXPERIMENTAL. Params sent with a request_user_input event.
type ToolRequestUserInputParams struct {
	// @deprecated Use `isBlocking` to decide whether the request should block.
	AutoResolutionMS *int64                               `json:"autoResolutionMs"`
	IsBlocking       bool                                 `json:"isBlocking"`
	ItemID           string                               `json:"itemId"`
	Questions        []ToolRequestUserInputParamsQuestion `json:"questions"`
	ThreadID         string                               `json:"threadId"`
	TurnID           string                               `json:"turnId"`
}

// EXPERIMENTAL. Represents one request_user_input question and its required options.
type ToolRequestUserInputParamsQuestion struct {
	Header   string                             `json:"header"`
	ID       string                             `json:"id"`
	IsOther  *bool                              `json:"isOther,omitempty"`
	IsSecret *bool                              `json:"isSecret,omitempty"`
	Options  []FluffyToolRequestUserInputOption `json:"options"`
	Question string                             `json:"question"`
}

// EXPERIMENTAL. Defines a single selectable option for request_user_input.
type FluffyToolRequestUserInputOption struct {
	Description string `json:"description"`
	Label       string `json:"label"`
}

// EXPERIMENTAL. Response payload mapping question ids to answers.
type ToolRequestUserInputResponse struct {
	Answers map[string]ToolRequestUserInputAnswer `json:"answers"`
}

// EXPERIMENTAL. Captures a user's answer to a request_user_input question.
type ToolRequestUserInputAnswer struct {
	Answers []string `json:"answers"`
}

type Type string

const (
	Add    Type = "add"
	Delete Type = "delete"
	Update Type = "update"
)

type NetworkPolicyRuleAction string

const (
	Allow                       NetworkPolicyRuleAction = "allow"
	NetworkPolicyRuleActionDeny NetworkPolicyRuleAction = "deny"
)

// User has approved this command and the agent should execute it.
//
// User has approved this request and wants future prompts in the same session-scoped
// approval cache to be automatically approved for the remainder of the session.
//
// User has approved this MCP tool call and wants to amend its policy so matching future
// calls are automatically approved across sessions.
//
// Automatic approval review timed out before reaching a decision.
//
// User has denied this command and the agent should not do anything until the user's next
// command.
type ReviewDecision string

const (
	Abort                      ReviewDecision = "abort"
	ApprovedForSession         ReviewDecision = "approved_for_session"
	ApprovedMCPPolicyAmendment ReviewDecision = "approved_mcp_policy_amendment"
	ReviewDecisionApproved     ReviewDecision = "approved"
	TimedOut                   ReviewDecision = "timed_out"
)

// Codex attempted a backend request and received `401 Unauthorized`.
type ChatgptAuthTokensRefreshReason string

const (
	ChatgptAuthTokensRefreshReasonUnauthorized ChatgptAuthTokensRefreshReason = "unauthorized"
)

type InitializedNotificationMethod string

const (
	Initialized InitializedNotificationMethod = "initialized"
)

type ClientRequestMethod string

const (
	AccountLoginCancel                     ClientRequestMethod = "account/login/cancel"
	AccountLoginStart                      ClientRequestMethod = "account/login/start"
	AccountLogout                          ClientRequestMethod = "account/logout"
	AccountRateLimitResetCreditConsume     ClientRequestMethod = "account/rateLimitResetCredit/consume"
	AccountRateLimitsRead                  ClientRequestMethod = "account/rateLimits/read"
	AccountRead                            ClientRequestMethod = "account/read"
	AccountSendAddCreditsNudgeEmail        ClientRequestMethod = "account/sendAddCreditsNudgeEmail"
	AccountUsageRead                       ClientRequestMethod = "account/usage/read"
	AccountWorkspaceMessagesRead           ClientRequestMethod = "account/workspaceMessages/read"
	AppInstalled                           ClientRequestMethod = "app/installed"
	AppList                                ClientRequestMethod = "app/list"
	AppRead                                ClientRequestMethod = "app/read"
	CommandExec                            ClientRequestMethod = "command/exec"
	CommandExecResize                      ClientRequestMethod = "command/exec/resize"
	CommandExecTerminate                   ClientRequestMethod = "command/exec/terminate"
	CommandExecWrite                       ClientRequestMethod = "command/exec/write"
	ConfigBatchWrite                       ClientRequestMethod = "config/batchWrite"
	ConfigMCPServerReload                  ClientRequestMethod = "config/mcpServer/reload"
	ConfigRead                             ClientRequestMethod = "config/read"
	ConfigRequirementsRead                 ClientRequestMethod = "configRequirements/read"
	ConfigValueWrite                       ClientRequestMethod = "config/value/write"
	ExperimentalFeatureEnablementSet       ClientRequestMethod = "experimentalFeature/enablement/set"
	ExperimentalFeatureList                ClientRequestMethod = "experimentalFeature/list"
	ExternalAgentConfigDetect              ClientRequestMethod = "externalAgentConfig/detect"
	ExternalAgentConfigImport              ClientRequestMethod = "externalAgentConfig/import"
	ExternalAgentConfigImportReadHistories ClientRequestMethod = "externalAgentConfig/import/readHistories"
	ExternalAgentConfigImportRecordHistory ClientRequestMethod = "externalAgentConfig/import/recordHistory"
	FSCopy                                 ClientRequestMethod = "fs/copy"
	FSCreateDirectory                      ClientRequestMethod = "fs/createDirectory"
	FSGetMetadata                          ClientRequestMethod = "fs/getMetadata"
	FSReadDirectory                        ClientRequestMethod = "fs/readDirectory"
	FSReadFile                             ClientRequestMethod = "fs/readFile"
	FSRemove                               ClientRequestMethod = "fs/remove"
	FSUnwatch                              ClientRequestMethod = "fs/unwatch"
	FSWatch                                ClientRequestMethod = "fs/watch"
	FSWriteFile                            ClientRequestMethod = "fs/writeFile"
	FeedbackUpload                         ClientRequestMethod = "feedback/upload"
	FuzzyFileSearch                        ClientRequestMethod = "fuzzyFileSearch"
	HooksList                              ClientRequestMethod = "hooks/list"
	Initialize                             ClientRequestMethod = "initialize"
	MCPServerOauthLogin                    ClientRequestMethod = "mcpServer/oauth/login"
	MCPServerResourceRead                  ClientRequestMethod = "mcpServer/resource/read"
	MCPServerStatusList                    ClientRequestMethod = "mcpServerStatus/list"
	MCPServerToolCall                      ClientRequestMethod = "mcpServer/tool/call"
	MarketplaceAdd                         ClientRequestMethod = "marketplace/add"
	MarketplaceRemove                      ClientRequestMethod = "marketplace/remove"
	MarketplaceUpgrade                     ClientRequestMethod = "marketplace/upgrade"
	ModelList                              ClientRequestMethod = "model/list"
	ModelProviderCapabilitiesRead          ClientRequestMethod = "modelProvider/capabilities/read"
	PermissionProfileList                  ClientRequestMethod = "permissionProfile/list"
	PluginInstall                          ClientRequestMethod = "plugin/install"
	PluginInstalled                        ClientRequestMethod = "plugin/installed"
	PluginList                             ClientRequestMethod = "plugin/list"
	PluginRead                             ClientRequestMethod = "plugin/read"
	PluginShareCheckout                    ClientRequestMethod = "plugin/share/checkout"
	PluginShareDelete                      ClientRequestMethod = "plugin/share/delete"
	PluginShareList                        ClientRequestMethod = "plugin/share/list"
	PluginShareSave                        ClientRequestMethod = "plugin/share/save"
	PluginShareUpdateTargets               ClientRequestMethod = "plugin/share/updateTargets"
	PluginSkillRead                        ClientRequestMethod = "plugin/skill/read"
	PluginUninstall                        ClientRequestMethod = "plugin/uninstall"
	ReviewStart                            ClientRequestMethod = "review/start"
	SkillsConfigWrite                      ClientRequestMethod = "skills/config/write"
	SkillsExtraRootsSet                    ClientRequestMethod = "skills/extraRoots/set"
	SkillsList                             ClientRequestMethod = "skills/list"
	ThreadApproveGuardianDeniedAction      ClientRequestMethod = "thread/approveGuardianDeniedAction"
	ThreadArchive                          ClientRequestMethod = "thread/archive"
	ThreadCompactStart                     ClientRequestMethod = "thread/compact/start"
	ThreadDelete                           ClientRequestMethod = "thread/delete"
	ThreadFork                             ClientRequestMethod = "thread/fork"
	ThreadGoalClear                        ClientRequestMethod = "thread/goal/clear"
	ThreadGoalGet                          ClientRequestMethod = "thread/goal/get"
	ThreadGoalSet                          ClientRequestMethod = "thread/goal/set"
	ThreadInjectItems                      ClientRequestMethod = "thread/inject_items"
	ThreadList                             ClientRequestMethod = "thread/list"
	ThreadLoadedList                       ClientRequestMethod = "thread/loaded/list"
	ThreadMetadataUpdate                   ClientRequestMethod = "thread/metadata/update"
	ThreadNameSet                          ClientRequestMethod = "thread/name/set"
	ThreadRead                             ClientRequestMethod = "thread/read"
	ThreadResume                           ClientRequestMethod = "thread/resume"
	ThreadRollback                         ClientRequestMethod = "thread/rollback"
	ThreadSectionCreate                    ClientRequestMethod = "threadSection/create"
	ThreadSectionDelete                    ClientRequestMethod = "threadSection/delete"
	ThreadSectionList                      ClientRequestMethod = "threadSection/list"
	ThreadSectionMove                      ClientRequestMethod = "thread/section/move"
	ThreadSectionUpdate                    ClientRequestMethod = "threadSection/update"
	ThreadShellCommand                     ClientRequestMethod = "thread/shellCommand"
	ThreadStart                            ClientRequestMethod = "thread/start"
	ThreadUnarchive                        ClientRequestMethod = "thread/unarchive"
	ThreadUnsubscribe                      ClientRequestMethod = "thread/unsubscribe"
	TurnInterrupt                          ClientRequestMethod = "turn/interrupt"
	TurnStart                              ClientRequestMethod = "turn/start"
	TurnSteer                              ClientRequestMethod = "turn/steer"
	WindowsSandboxReadiness                ClientRequestMethod = "windowsSandbox/readiness"
	WindowsSandboxSetupStart               ClientRequestMethod = "windowsSandbox/setupStart"
)

type LoginAppBrand string

const (
	Codex                LoginAppBrand = "codex"
	LoginAppBrandChatgpt LoginAppBrand = "chatgpt"
)

type ApprovalPolicyEnum string

const (
	Never     ApprovalPolicyEnum = "never"
	OnRequest ApprovalPolicyEnum = "on-request"
	Untrusted ApprovalPolicyEnum = "untrusted"
)

// Configures who approval requests are routed to for review. Examples include sandbox
// escapes, blocked network access, MCP approval prompts, and ARC escalations. Defaults to
// `user`. `auto_review` uses a carefully prompted subagent to gather relevant context and
// apply a risk-based decision framework before approving or denying the request. The legacy
// value `guardian_subagent` is accepted for compatibility.
type ApprovalsReviewer string

const (
	ApprovalsReviewerUser ApprovalsReviewer = "user"
	AutoReview            ApprovalsReviewer = "auto_review"
	GuardianSubagent      ApprovalsReviewer = "guardian_subagent"
)

type MCPServerOauthClientRegistration string

const (
	Cimd                                 MCPServerOauthClientRegistration = "cimd"
	Dcr                                  MCPServerOauthClientRegistration = "dcr"
	MCPServerOauthClientRegistrationAuto MCPServerOauthClientRegistration = "auto"
)

type AddCreditsNudgeCreditType string

const (
	Credits    AddCreditsNudgeCreditType = "credits"
	UsageLimit AddCreditsNudgeCreditType = "usage_limit"
)

type ReviewDelivery string

const (
	Detached ReviewDelivery = "detached"
	Inline   ReviewDelivery = "inline"
)

type MCPServerStatusDetail string

const (
	MCPServerStatusDetailFull MCPServerStatusDetail = "full"
	ToolsAndAuthOnly          MCPServerStatusDetail = "toolsAndAuthOnly"
)

type PluginShareEDiscoverability string

const (
	Listed   PluginShareEDiscoverability = "LISTED"
	Private  PluginShareEDiscoverability = "PRIVATE"
	Unlisted PluginShareEDiscoverability = "UNLISTED"
)

type MergeStrategy string

const (
	Replace MergeStrategy = "replace"
	Upsert  MergeStrategy = "upsert"
)

type ImageDetail string

const (
	ImageDetailAuto ImageDetail = "auto"
	ImageDetailHigh ImageDetail = "high"
	ImageDetailLow  ImageDetail = "low"
	Original        ImageDetail = "original"
)

type UserInputType string

const (
	Audio      UserInputType = "audio"
	Image      UserInputType = "image"
	LocalAudio UserInputType = "localAudio"
	LocalImage UserInputType = "localImage"
	Mention    UserInputType = "mention"
	Skill      UserInputType = "skill"
	Text       UserInputType = "text"
)

type ExternalAgentConfigMigrationItemType string

const (
	AgentsMd        ExternalAgentConfigMigrationItemType = "AGENTS_MD"
	Commands        ExternalAgentConfigMigrationItemType = "COMMANDS"
	Config          ExternalAgentConfigMigrationItemType = "CONFIG"
	Hooks           ExternalAgentConfigMigrationItemType = "HOOKS"
	MCPServerConfig ExternalAgentConfigMigrationItemType = "MCP_SERVER_CONFIG"
	Memory          ExternalAgentConfigMigrationItemType = "MEMORY"
	Plugins         ExternalAgentConfigMigrationItemType = "PLUGINS"
	Sessions        ExternalAgentConfigMigrationItemType = "SESSIONS"
	Skills          ExternalAgentConfigMigrationItemType = "SKILLS"
	Subagents       ExternalAgentConfigMigrationItemType = "SUBAGENTS"
)

type PluginListMarketplaceKind string

const (
	CreatedByMeRemote  PluginListMarketplaceKind = "created-by-me-remote"
	Local              PluginListMarketplaceKind = "local"
	SharedWithMe       PluginListMarketplaceKind = "shared-with-me"
	Vertical           PluginListMarketplaceKind = "vertical"
	WorkspaceDirectory PluginListMarketplaceKind = "workspace-directory"
)

type WindowsSandboxSetupMode string

const (
	Elevated   WindowsSandboxSetupMode = "elevated"
	Unelevated WindowsSandboxSetupMode = "unelevated"
)

type Personality string

const (
	Friendly        Personality = "friendly"
	PersonalityNone Personality = "none"
	Pragmatic       Personality = "pragmatic"
)

type SandboxMode string

const (
	DangerFullAccess SandboxMode = "danger-full-access"
	ReadOnly         SandboxMode = "read-only"
	WorkspaceWrite   SandboxMode = "workspace-write"
)

type NetworkAccess string

const (
	Enabled    NetworkAccess = "enabled"
	Restricted NetworkAccess = "restricted"
)

type SandboxPolicyType string

const (
	ExternalSandbox                   SandboxPolicyType = "externalSandbox"
	SandboxPolicyTypeDangerFullAccess SandboxPolicyType = "dangerFullAccess"
	SandboxPolicyTypeReadOnly         SandboxPolicyType = "readOnly"
	SandboxPolicyTypeWorkspaceWrite   SandboxPolicyType = "workspaceWrite"
)

type ThreadStartSource string

const (
	Clear   ThreadStartSource = "clear"
	Startup ThreadStartSource = "startup"
)

type PluginSharePrincipalType string

const (
	Group                        PluginSharePrincipalType = "group"
	PluginSharePrincipalTypeUser PluginSharePrincipalType = "user"
	Workspace                    PluginSharePrincipalType = "workspace"
)

type PluginShareTargetRole string

const (
	Editor PluginShareTargetRole = "editor"
	Reader PluginShareTargetRole = "reader"
)

type SortDirection string

const (
	Asc  SortDirection = "asc"
	Desc SortDirection = "desc"
)

type ThreadSortKey string

const (
	CreatedAt       ThreadSortKey = "created_at"
	RecencyAt       ThreadSortKey = "recency_at"
	SectionPosition ThreadSortKey = "section_position"
	UpdatedAt       ThreadSortKey = "updated_at"
)

type ThreadSourceKind string

const (
	SubAgent                  ThreadSourceKind = "subAgent"
	SubAgentCompact           ThreadSourceKind = "subAgentCompact"
	SubAgentOther             ThreadSourceKind = "subAgentOther"
	SubAgentReview            ThreadSourceKind = "subAgentReview"
	SubAgentThreadSpawn       ThreadSourceKind = "subAgentThreadSpawn"
	ThreadSourceKindAppServer ThreadSourceKind = "appServer"
	ThreadSourceKindCLI       ThreadSourceKind = "cli"
	ThreadSourceKindExec      ThreadSourceKind = "exec"
	ThreadSourceKindUnknown   ThreadSourceKind = "unknown"
	ThreadSourceKindVscode    ThreadSourceKind = "vscode"
)

type ThreadGoalStatus string

const (
	BudgetLimited           ThreadGoalStatus = "budgetLimited"
	Complete                ThreadGoalStatus = "complete"
	Paused                  ThreadGoalStatus = "paused"
	ThreadGoalStatusActive  ThreadGoalStatus = "active"
	ThreadGoalStatusBlocked ThreadGoalStatus = "blocked"
	UsageLimited            ThreadGoalStatus = "usageLimited"
)

// Option to disable reasoning summaries.
type ReasoningSummary string

const (
	Concise              ReasoningSummary = "concise"
	Detailed             ReasoningSummary = "detailed"
	ReasoningSummaryAuto ReasoningSummary = "auto"
	ReasoningSummaryNone ReasoningSummary = "none"
)

type ReviewTargetType string

const (
	BaseBranch         ReviewTargetType = "baseBranch"
	Commit             ReviewTargetType = "commit"
	Custom             ReviewTargetType = "custom"
	UncommittedChanges ReviewTargetType = "uncommittedChanges"
)

type LoginAccountParamsType string

const (
	APIKey                                  LoginAccountParamsType = "apiKey"
	AmazonBedrock                           LoginAccountParamsType = "amazonBedrock"
	ChatgptDeviceCode                       LoginAccountParamsType = "chatgptDeviceCode"
	LoginAccountParamsTypeChatgpt           LoginAccountParamsType = "chatgpt"
	LoginAccountParamsTypeChatgptAuthTokens LoginAccountParamsType = "chatgptAuthTokens"
)

type CommandActionType string

const (
	CommandActionTypeRead    CommandActionType = "read"
	CommandActionTypeSearch  CommandActionType = "search"
	CommandActionTypeUnknown CommandActionType = "unknown"
	ListFiles                CommandActionType = "listFiles"
)

type NetworkApprovalProtocol string

const (
	HTTP      NetworkApprovalProtocol = "http"
	HTTPS     NetworkApprovalProtocol = "https"
	Socks5TCP NetworkApprovalProtocol = "socks5Tcp"
	Socks5UDP NetworkApprovalProtocol = "socks5Udp"
)

// User approved the command.
//
// User approved the command and future prompts in the same session-scoped approval cache
// should run without prompting.
//
// User denied the command. The agent will continue the turn.
//
// User denied the command. The turn will also be immediately interrupted.
//
// User approved the file changes.
//
// User approved the file changes and future changes to the same files should run without
// prompting.
//
// User denied the file changes. The agent will continue the turn.
//
// User denied the file changes. The turn will also be immediately interrupted.
type FileChangeApprovalDecision string

const (
	AcceptForSession                  FileChangeApprovalDecision = "acceptForSession"
	FileChangeApprovalDecisionAccept  FileChangeApprovalDecision = "accept"
	FileChangeApprovalDecisionCancel  FileChangeApprovalDecision = "cancel"
	FileChangeApprovalDecisionDecline FileChangeApprovalDecision = "decline"
)

type InputDynamicToolCallOutputContentItemType string

const (
	InputAudio InputDynamicToolCallOutputContentItemType = "inputAudio"
	InputImage InputDynamicToolCallOutputContentItemType = "inputImage"
	InputText  InputDynamicToolCallOutputContentItemType = "inputText"
)

type ParsedCommandType string

const (
	ParsedCommandTypeListFiles ParsedCommandType = "list_files"
	ParsedCommandTypeRead      ParsedCommandType = "read"
	ParsedCommandTypeSearch    ParsedCommandType = "search"
	ParsedCommandTypeUnknown   ParsedCommandType = "unknown"
)

type FuzzyFileSearchMatchType string

const (
	Directory FuzzyFileSearchMatchType = "directory"
	File      FuzzyFileSearchMatchType = "file"
)

type Mode string

const (
	Form       Mode = "form"
	OpenaiForm Mode = "openai/form"
	URL        Mode = "url"
)

type MCPServerElicitationAction string

const (
	MCPServerElicitationActionAccept  MCPServerElicitationAction = "accept"
	MCPServerElicitationActionCancel  MCPServerElicitationAction = "cancel"
	MCPServerElicitationActionDecline MCPServerElicitationAction = "decline"
)

type FileSystemAccessMode string

const (
	FileSystemAccessModeDeny FileSystemAccessMode = "deny"
	FileSystemAccessModeRead FileSystemAccessMode = "read"
	Write                    FileSystemAccessMode = "write"
)

type FileSystemPathType string

const (
	GlobPattern FileSystemPathType = "glob_pattern"
	Path        FileSystemPathType = "path"
	Special     FileSystemPathType = "special"
)

type Kind string

const (
	KindUnknown  Kind = "unknown"
	Minimal      Kind = "minimal"
	ProjectRoots Kind = "project_roots"
	Root         Kind = "root"
	SlashTmp     Kind = "slash_tmp"
	Tmpdir       Kind = "tmpdir"
)

type PermissionGrantScope string

const (
	PermissionGrantScopeTurn PermissionGrantScope = "turn"
	Session                  PermissionGrantScope = "session"
)

type NotificationMethod string

const (
	AccountLoginCompleted                   NotificationMethod = "account/login/completed"
	AccountRateLimitsUpdated                NotificationMethod = "account/rateLimits/updated"
	AccountUpdated                          NotificationMethod = "account/updated"
	AppListUpdated                          NotificationMethod = "app/list/updated"
	AutoApprovalReviewStrictReviewRequired  NotificationMethod = "autoApprovalReview/strictReviewRequired"
	CommandExecOutputDelta                  NotificationMethod = "command/exec/outputDelta"
	ConfigWarning                           NotificationMethod = "configWarning"
	DeprecationNotice                       NotificationMethod = "deprecationNotice"
	ExternalAgentConfigImportCompleted      NotificationMethod = "externalAgentConfig/import/completed"
	ExternalAgentConfigImportProgress       NotificationMethod = "externalAgentConfig/import/progress"
	FSChanged                               NotificationMethod = "fs/changed"
	FuzzyFileSearchSessionCompleted         NotificationMethod = "fuzzyFileSearch/sessionCompleted"
	FuzzyFileSearchSessionUpdated           NotificationMethod = "fuzzyFileSearch/sessionUpdated"
	GuardianWarning                         NotificationMethod = "guardianWarning"
	HookCompleted                           NotificationMethod = "hook/completed"
	HookStarted                             NotificationMethod = "hook/started"
	ItemAgentMessageDelta                   NotificationMethod = "item/agentMessage/delta"
	ItemAutoApprovalReviewCompleted         NotificationMethod = "item/autoApprovalReview/completed"
	ItemAutoApprovalReviewStarted           NotificationMethod = "item/autoApprovalReview/started"
	ItemCommandExecutionOutputDelta         NotificationMethod = "item/commandExecution/outputDelta"
	ItemCommandExecutionTerminalInteraction NotificationMethod = "item/commandExecution/terminalInteraction"
	ItemCompleted                           NotificationMethod = "item/completed"
	ItemFileChangeOutputDelta               NotificationMethod = "item/fileChange/outputDelta"
	ItemFileChangePatchUpdated              NotificationMethod = "item/fileChange/patchUpdated"
	ItemMCPToolCallProgress                 NotificationMethod = "item/mcpToolCall/progress"
	ItemPlanDelta                           NotificationMethod = "item/plan/delta"
	ItemReasoningSummaryPartAdded           NotificationMethod = "item/reasoning/summaryPartAdded"
	ItemReasoningSummaryTextDelta           NotificationMethod = "item/reasoning/summaryTextDelta"
	ItemReasoningTextDelta                  NotificationMethod = "item/reasoning/textDelta"
	ItemStarted                             NotificationMethod = "item/started"
	MCPServerOauthLoginCompleted            NotificationMethod = "mcpServer/oauthLogin/completed"
	MCPServerStartupStatusUpdated           NotificationMethod = "mcpServer/startupStatus/updated"
	ModelRerouted                           NotificationMethod = "model/rerouted"
	ModelSafetyBufferingUpdated             NotificationMethod = "model/safetyBuffering/updated"
	ModelVerification                       NotificationMethod = "model/verification"
	NotificationMethodError                 NotificationMethod = "error"
	NotificationMethodWarning               NotificationMethod = "warning"
	ProcessExited                           NotificationMethod = "process/exited"
	ProcessOutputDelta                      NotificationMethod = "process/outputDelta"
	ProjectChanged                          NotificationMethod = "project/changed"
	RemoteControlStatusChanged              NotificationMethod = "remoteControl/status/changed"
	ServerRequestResolved                   NotificationMethod = "serverRequest/resolved"
	SkillsChanged                           NotificationMethod = "skills/changed"
	ThreadArchived                          NotificationMethod = "thread/archived"
	ThreadClosed                            NotificationMethod = "thread/closed"
	ThreadCompacted                         NotificationMethod = "thread/compacted"
	ThreadDeleted                           NotificationMethod = "thread/deleted"
	ThreadEnvironmentConnected              NotificationMethod = "thread/environment/connected"
	ThreadEnvironmentDisconnected           NotificationMethod = "thread/environment/disconnected"
	ThreadGoalCleared                       NotificationMethod = "thread/goal/cleared"
	ThreadGoalUpdated                       NotificationMethod = "thread/goal/updated"
	ThreadNameUpdated                       NotificationMethod = "thread/name/updated"
	ThreadProjectUpdated                    NotificationMethod = "thread/project/updated"
	ThreadQueueChanged                      NotificationMethod = "thread/queue/changed"
	ThreadRealtimeClosed                    NotificationMethod = "thread/realtime/closed"
	ThreadRealtimeError                     NotificationMethod = "thread/realtime/error"
	ThreadRealtimeItemAdded                 NotificationMethod = "thread/realtime/itemAdded"
	ThreadRealtimeOutputAudioDelta          NotificationMethod = "thread/realtime/outputAudio/delta"
	ThreadRealtimeSDP                       NotificationMethod = "thread/realtime/sdp"
	ThreadRealtimeStarted                   NotificationMethod = "thread/realtime/started"
	ThreadRealtimeTranscriptDelta           NotificationMethod = "thread/realtime/transcript/delta"
	ThreadRealtimeTranscriptDone            NotificationMethod = "thread/realtime/transcript/done"
	ThreadReverted                          NotificationMethod = "thread/reverted"
	ThreadSettingsUpdated                   NotificationMethod = "thread/settings/updated"
	ThreadStarted                           NotificationMethod = "thread/started"
	ThreadStatusChanged                     NotificationMethod = "thread/status/changed"
	ThreadTokenUsageUpdated                 NotificationMethod = "thread/tokenUsage/updated"
	ThreadUnarchived                        NotificationMethod = "thread/unarchived"
	TurnCompleted                           NotificationMethod = "turn/completed"
	TurnDiffUpdated                         NotificationMethod = "turn/diff/updated"
	TurnModerationMetadata                  NotificationMethod = "turn/moderationMetadata"
	TurnPlanUpdated                         NotificationMethod = "turn/plan/updated"
	TurnStarted                             NotificationMethod = "turn/started"
	WindowsSandboxSetupCompleted            NotificationMethod = "windowsSandbox/setupCompleted"
	WindowsWorldWritableWarning             NotificationMethod = "windows/worldWritableWarning"
)

type GuardianCommandSource string

const (
	Shell       GuardianCommandSource = "shell"
	UnifiedExec GuardianCommandSource = "unifiedExec"
)

type GuardianApprovalReviewActionType string

const (
	ApplyPatch                                    GuardianApprovalReviewActionType = "applyPatch"
	Execve                                        GuardianApprovalReviewActionType = "execve"
	GuardianApprovalReviewActionTypeCommand       GuardianApprovalReviewActionType = "command"
	GuardianApprovalReviewActionTypeMCPToolCall   GuardianApprovalReviewActionType = "mcpToolCall"
	GuardianApprovalReviewActionTypeNetworkAccess GuardianApprovalReviewActionType = "networkAccess"
	RequestPermissions                            GuardianApprovalReviewActionType = "requestPermissions"
)

// OpenAI API key provided by the caller and stored by Codex.
//
// ChatGPT OAuth managed by Codex (tokens persisted and refreshed by Codex).
//
// [UNSTABLE] FOR OPENAI INTERNAL USE ONLY - DO NOT USE.
//
// ChatGPT auth tokens are supplied by an external host app and are only stored in memory.
// Token refresh must be handled by the external host app.
//
// Backend auth supplied as request headers.
//
// Programmatic Codex auth backed by a registered Agent Identity.
//
// Programmatic Codex auth backed by a personal access token.
//
// Amazon Bedrock bearer token managed by Codex.
type AuthMode string

const (
	AgentIdentity             AuthMode = "agentIdentity"
	Apikey                    AuthMode = "apikey"
	AuthModeChatgpt           AuthMode = "chatgpt"
	AuthModeChatgptAuthTokens AuthMode = "chatgptAuthTokens"
	BedrockAPIKey             AuthMode = "bedrockApiKey"
	Headers                   AuthMode = "headers"
	PersonalAccessToken       AuthMode = "personalAccessToken"
)

type ProjectChangeType string

const (
	Created ProjectChangeType = "created"
	Deleted ProjectChangeType = "deleted"
	Updated ProjectChangeType = "updated"
)

// [UNSTABLE] Source that produced a terminal approval auto-review decision.
type AutoReviewDecisionSource string

const (
	AutoReviewDecisionSourceAgent AutoReviewDecisionSource = "agent"
)

type NonSteerableTurnKind string

const (
	NonSteerableTurnKindCompact NonSteerableTurnKind = "compact"
	NonSteerableTurnKindReview  NonSteerableTurnKind = "review"
)

type CodexErrorInfoEnum string

const (
	BadRequest                       CodexErrorInfoEnum = "badRequest"
	CodexErrorInfoOther              CodexErrorInfoEnum = "other"
	CodexErrorInfoUnauthorized       CodexErrorInfoEnum = "unauthorized"
	CodexErrorInfoUsageLimitExceeded CodexErrorInfoEnum = "usageLimitExceeded"
	ContextWindowExceeded            CodexErrorInfoEnum = "contextWindowExceeded"
	CyberPolicy                      CodexErrorInfoEnum = "cyberPolicy"
	InternalServerError              CodexErrorInfoEnum = "internalServerError"
	MisalignmentPolicyViolation      CodexErrorInfoEnum = "misalignmentPolicyViolation"
	SandboxError                     CodexErrorInfoEnum = "sandboxError"
	ServerOverloaded                 CodexErrorInfoEnum = "serverOverloaded"
	SessionBudgetExceeded            CodexErrorInfoEnum = "sessionBudgetExceeded"
	ThreadRollbackFailed             CodexErrorInfoEnum = "threadRollbackFailed"
)

type MCPServerStartupFailureReason string

const (
	ReauthenticationRequired MCPServerStartupFailureReason = "reauthenticationRequired"
)

type DesktopOnboardingEntrypoint string

const (
	LifeSciences DesktopOnboardingEntrypoint = "life_sciences"
)

type TurnPlanStepStatus string

const (
	Pending                      TurnPlanStepStatus = "pending"
	TurnPlanStepStatusCompleted  TurnPlanStepStatus = "completed"
	TurnPlanStepStatusInProgress TurnPlanStepStatus = "inProgress"
)

type PlanType string

const (
	Business                    PlanType = "business"
	Edu                         PlanType = "edu"
	EduPlus                     PlanType = "edu_plus"
	EduPro                      PlanType = "edu_pro"
	Ent26                       PlanType = "ent26"
	Enterprise                  PlanType = "enterprise"
	EnterpriseCbpAutomation     PlanType = "enterprise_cbp_automation"
	EnterpriseCbpUsageBased     PlanType = "enterprise_cbp_usage_based"
	Free                        PlanType = "free"
	Go                          PlanType = "go"
	PlanTypeUnknown             PlanType = "unknown"
	Plus                        PlanType = "plus"
	Pro                         PlanType = "pro"
	Prolite                     PlanType = "prolite"
	SelfServeBusinessProlite    PlanType = "self_serve_business_prolite"
	SelfServeBusinessUsageBased PlanType = "self_serve_business_usage_based"
	Team                        PlanType = "team"
)

type RateLimitReachedType string

const (
	RateLimitReached                 RateLimitReachedType = "rate_limit_reached"
	WorkspaceMemberCreditsDepleted   RateLimitReachedType = "workspace_member_credits_depleted"
	WorkspaceMemberUsageLimitReached RateLimitReachedType = "workspace_member_usage_limit_reached"
	WorkspaceOwnerCreditsDepleted    RateLimitReachedType = "workspace_owner_credits_depleted"
	WorkspaceOwnerUsageLimitReached  RateLimitReachedType = "workspace_owner_usage_limit_reached"
)

// [UNSTABLE] Risk level assigned by approval auto-review.
type GuardianRiskLevel string

const (
	Critical                GuardianRiskLevel = "critical"
	GuardianRiskLevelHigh   GuardianRiskLevel = "high"
	GuardianRiskLevelLow    GuardianRiskLevel = "low"
	GuardianRiskLevelMedium GuardianRiskLevel = "medium"
)

// [UNSTABLE] Lifecycle state for an approval auto-review.
type GuardianApprovalReviewStatus string

const (
	Aborted                                GuardianApprovalReviewStatus = "aborted"
	Denied                                 GuardianApprovalReviewStatus = "denied"
	GuardianApprovalReviewStatusApproved   GuardianApprovalReviewStatus = "approved"
	GuardianApprovalReviewStatusInProgress GuardianApprovalReviewStatus = "inProgress"
	GuardianApprovalReviewStatusTimedOut   GuardianApprovalReviewStatus = "timedOut"
)

// [UNSTABLE] Authorization level assigned by approval auto-review.
type GuardianUserAuthorization string

const (
	GuardianUserAuthorizationHigh    GuardianUserAuthorization = "high"
	GuardianUserAuthorizationLow     GuardianUserAuthorization = "low"
	GuardianUserAuthorizationMedium  GuardianUserAuthorization = "medium"
	GuardianUserAuthorizationUnknown GuardianUserAuthorization = "unknown"
)

type HookOutputEntryKind string

const (
	Context                    HookOutputEntryKind = "context"
	Feedback                   HookOutputEntryKind = "feedback"
	HookOutputEntryKindError   HookOutputEntryKind = "error"
	HookOutputEntryKindStop    HookOutputEntryKind = "stop"
	HookOutputEntryKindWarning HookOutputEntryKind = "warning"
)

type HookEventName string

const (
	HookEventNameStop HookEventName = "stop"
	PermissionRequest HookEventName = "permissionRequest"
	PostCompact       HookEventName = "postCompact"
	PostToolUse       HookEventName = "postToolUse"
	PreCompact        HookEventName = "preCompact"
	PreToolUse        HookEventName = "preToolUse"
	SessionEnd        HookEventName = "sessionEnd"
	SessionStart      HookEventName = "sessionStart"
	SubagentStart     HookEventName = "subagentStart"
	SubagentStop      HookEventName = "subagentStop"
	UserPromptSubmit  HookEventName = "userPromptSubmit"
)

type HookExecutionMode string

const (
	HookExecutionModeAsync HookExecutionMode = "async"
	Sync                   HookExecutionMode = "sync"
)

type HookHandlerType string

const (
	HookHandlerTypeAgent   HookHandlerType = "agent"
	HookHandlerTypeCommand HookHandlerType = "command"
	MCPTool                HookHandlerType = "mcpTool"
	Prompt                 HookHandlerType = "prompt"
)

type HookScope string

const (
	HookScopeTurn HookScope = "turn"
	Thread        HookScope = "thread"
)

type HookSource string

const (
	CloudManagedConfig      HookSource = "cloudManagedConfig"
	CloudRequirements       HookSource = "cloudRequirements"
	HookSourceUnknown       HookSource = "unknown"
	HookSourceUser          HookSource = "user"
	LegacyManagedConfigFile HookSource = "legacyManagedConfigFile"
	LegacyManagedConfigMdm  HookSource = "legacyManagedConfigMdm"
	Mdm                     HookSource = "mdm"
	Plugin                  HookSource = "plugin"
	Project                 HookSource = "project"
	SessionFlags            HookSource = "sessionFlags"
	System                  HookSource = "system"
)

type HookRunStatus string

const (
	HookRunStatusBlocked   HookRunStatus = "blocked"
	HookRunStatusCompleted HookRunStatus = "completed"
	HookRunStatusFailed    HookRunStatus = "failed"
	HookRunStatusRunning   HookRunStatus = "running"
	Stopped                HookRunStatus = "stopped"
)

type ThreadActiveFlag string

const (
	WaitingOnApproval  ThreadActiveFlag = "waitingOnApproval"
	WaitingOnUserInput ThreadActiveFlag = "waitingOnUserInput"
)

type ThreadStatusType string

const (
	Idle                      ThreadStatusType = "idle"
	SystemError               ThreadStatusType = "systemError"
	ThreadStatusTypeActive    ThreadStatusType = "active"
	ThreadStatusTypeNotLoaded ThreadStatusType = "notLoaded"
)

type MCPServerStartupState string

const (
	Cancelled                    MCPServerStartupState = "cancelled"
	Connected                    MCPServerStartupState = "connected"
	Connecting                   MCPServerStartupState = "connecting"
	Disabled                     MCPServerStartupState = "disabled"
	MCPServerStartupStateErrored MCPServerStartupState = "errored"
	MCPServerStartupStateFailed  MCPServerStartupState = "failed"
	Ready                        MCPServerStartupState = "ready"
	Starting                     MCPServerStartupState = "starting"
)

// stdout stream. PTY mode multiplexes terminal output here.
//
// stderr stream.
type OutputStream string

const (
	Stderr OutputStream = "stderr"
	Stdout OutputStream = "stdout"
)

type SubAgentSourceEnum string

const (
	MemoryConsolidation   SubAgentSourceEnum = "memory_consolidation"
	SubAgentSourceCompact SubAgentSourceEnum = "compact"
	SubAgentSourceReview  SubAgentSourceEnum = "review"
)

type SessionSourceEnum string

const (
	SessionSourceAppServer SessionSourceEnum = "appServer"
	SessionSourceCLI       SessionSourceEnum = "cli"
	SessionSourceExec      SessionSourceEnum = "exec"
	SessionSourceUnknown   SessionSourceEnum = "unknown"
	SessionSourceVscode    SessionSourceEnum = "vscode"
)

type WebSearchActionType string

const (
	FindInPage                WebSearchActionType = "findInPage"
	OpenPage                  WebSearchActionType = "openPage"
	WebSearchActionTypeOther  WebSearchActionType = "other"
	WebSearchActionTypeSearch WebSearchActionType = "search"
)

type CollabAgentStatus string

const (
	CollabAgentStatusCompleted   CollabAgentStatus = "completed"
	CollabAgentStatusErrored     CollabAgentStatus = "errored"
	CollabAgentStatusInterrupted CollabAgentStatus = "interrupted"
	CollabAgentStatusRunning     CollabAgentStatus = "running"
	NotFound                     CollabAgentStatus = "notFound"
	PendingInit                  CollabAgentStatus = "pendingInit"
	Shutdown                     CollabAgentStatus = "shutdown"
)

type AgentMessageDelivery string

const (
	AgentMessageDeliveryAsync AgentMessageDelivery = "async"
)

type UsageLimitExceededImageGenerationFailureType string

const (
	UsageLimitExceededImageGenerationFailureTypeUsageLimitExceeded UsageLimitExceededImageGenerationFailureType = "usageLimitExceeded"
)

type SubAgentActivityKind string

const (
	Interacted                      SubAgentActivityKind = "interacted"
	Started                         SubAgentActivityKind = "started"
	SubAgentActivityKindInterrupted SubAgentActivityKind = "interrupted"
)

// Mid-turn assistant text (for example preamble/progress narration).
//
// Additional tool calls or assistant output may follow before turn completion.
//
// The assistant's terminal answer text for the current turn.
type MessagePhase string

const (
	Commentary  MessagePhase = "commentary"
	FinalAnswer MessagePhase = "final_answer"
)

type CommandExecutionSource string

const (
	CommandExecutionSourceAgent CommandExecutionSource = "agent"
	UnifiedExecInteraction      CommandExecutionSource = "unifiedExecInteraction"
	UnifiedExecStartup          CommandExecutionSource = "unifiedExecStartup"
	UserShell                   CommandExecutionSource = "userShell"
)

type ThreadItemType string

const (
	AgentMessage              ThreadItemType = "agentMessage"
	CollabAgentToolCall       ThreadItemType = "collabAgentToolCall"
	CommandExecution          ThreadItemType = "commandExecution"
	ContextCompaction         ThreadItemType = "contextCompaction"
	DynamicToolCall           ThreadItemType = "dynamicToolCall"
	EnteredReviewMode         ThreadItemType = "enteredReviewMode"
	ExitedReviewMode          ThreadItemType = "exitedReviewMode"
	FileChange                ThreadItemType = "fileChange"
	HookPrompt                ThreadItemType = "hookPrompt"
	ImageGeneration           ThreadItemType = "imageGeneration"
	ImageView                 ThreadItemType = "imageView"
	Reasoning                 ThreadItemType = "reasoning"
	Sleep                     ThreadItemType = "sleep"
	SubAgentActivity          ThreadItemType = "subAgentActivity"
	ThreadItemTypeMCPToolCall ThreadItemType = "mcpToolCall"
	ThreadItemTypePlan        ThreadItemType = "plan"
	UserMessage               ThreadItemType = "userMessage"
	WebSearch                 ThreadItemType = "webSearch"
)

// Describes how much of `items` has been loaded for this turn.
//
// `items` was not loaded for this turn. The field is intentionally empty.
//
// `items` contains only a display summary for this turn.
//
// `items` contains every ThreadItem available from persisted app-server history for this
// turn.
type TurnItemsView string

const (
	Summary                TurnItemsView = "summary"
	TurnItemsViewFull      TurnItemsView = "full"
	TurnItemsViewNotLoaded TurnItemsView = "notLoaded"
)

type TurnStatus string

const (
	TurnStatusCompleted   TurnStatus = "completed"
	TurnStatusFailed      TurnStatus = "failed"
	TurnStatusInProgress  TurnStatus = "inProgress"
	TurnStatusInterrupted TurnStatus = "interrupted"
)

// Initial collaboration mode to use when the TUI starts.
type ModeKind string

const (
	Default      ModeKind = "default"
	ModeKindPlan ModeKind = "plan"
)

type VerificationElement string

const (
	TrustedAccessForCyber VerificationElement = "trustedAccessForCyber"
)

type RealtimeConversationVersion string

const (
	V1 RealtimeConversationVersion = "v1"
	V2 RealtimeConversationVersion = "v2"
	V3 RealtimeConversationVersion = "v3"
)

type ServerRequestMethod string

const (
	AccountChatgptAuthTokensRefresh     ServerRequestMethod = "account/chatgptAuthTokens/refresh"
	ApplyPatchApproval                  ServerRequestMethod = "applyPatchApproval"
	AttestationGenerate                 ServerRequestMethod = "attestation/generate"
	ExecCommandApproval                 ServerRequestMethod = "execCommandApproval"
	ItemCommandExecutionRequestApproval ServerRequestMethod = "item/commandExecution/requestApproval"
	ItemFileChangeRequestApproval       ServerRequestMethod = "item/fileChange/requestApproval"
	ItemPermissionsRequestApproval      ServerRequestMethod = "item/permissions/requestApproval"
	ItemToolCall                        ServerRequestMethod = "item/tool/call"
	ItemToolRequestUserInput            ServerRequestMethod = "item/tool/requestUserInput"
	MCPServerElicitationRequest         ServerRequestMethod = "mcpServer/elicitation/request"
)

// User's decision in response to an ExecApprovalRequest.
type ApplyPatchApprovalResponseReviewDecision struct {
	Enum                 *ReviewDecision
	PurpleReviewDecision *PurpleReviewDecision
}

func (x *ApplyPatchApprovalResponseReviewDecision) UnmarshalJSON(data []byte) error {
	x.PurpleReviewDecision = nil
	x.Enum = nil
	var c PurpleReviewDecision
	object, err := unmarshalUnion(data, nil, nil, nil, nil, false, nil, true, &c, false, nil, true, &x.Enum, false)
	if err != nil {
		return err
	}
	if object {
		x.PurpleReviewDecision = &c
	}
	return nil
}

func (x *ApplyPatchApprovalResponseReviewDecision) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, nil, false, nil, x.PurpleReviewDecision != nil, x.PurpleReviewDecision, false, nil, x.Enum != nil, x.Enum, false)
}

type RequestID struct {
	Integer *int64
	String  *string
}

func (x *RequestID) UnmarshalJSON(data []byte) error {
	object, err := unmarshalUnion(data, &x.Integer, nil, nil, &x.String, false, nil, false, nil, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
	}
	return nil
}

func (x *RequestID) MarshalJSON() ([]byte, error) {
	return marshalUnion(x.Integer, nil, nil, x.String, false, nil, false, nil, false, nil, false, nil, false)
}

type ApprovalPolicyUnion struct {
	ApprovalPolicyGranularAskForApproval *ApprovalPolicyGranularAskForApproval
	Enum                                 *ApprovalPolicyEnum
}

func (x *ApprovalPolicyUnion) UnmarshalJSON(data []byte) error {
	x.ApprovalPolicyGranularAskForApproval = nil
	x.Enum = nil
	var c ApprovalPolicyGranularAskForApproval
	object, err := unmarshalUnion(data, nil, nil, nil, nil, false, nil, true, &c, false, nil, true, &x.Enum, true)
	if err != nil {
		return err
	}
	if object {
		x.ApprovalPolicyGranularAskForApproval = &c
	}
	return nil
}

func (x *ApprovalPolicyUnion) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, nil, false, nil, x.ApprovalPolicyGranularAskForApproval != nil, x.ApprovalPolicyGranularAskForApproval, false, nil, x.Enum != nil, x.Enum, true)
}

type Command struct {
	String      *string
	StringArray []string
}

func (x *Command) UnmarshalJSON(data []byte) error {
	x.StringArray = nil
	object, err := unmarshalUnion(data, nil, nil, nil, &x.String, true, &x.StringArray, false, nil, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
	}
	return nil
}

func (x *Command) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, x.String, x.StringArray != nil, x.StringArray, false, nil, false, nil, false, nil, false)
}

type ThreadListCwdFilter struct {
	String      *string
	StringArray []string
}

func (x *ThreadListCwdFilter) UnmarshalJSON(data []byte) error {
	x.StringArray = nil
	object, err := unmarshalUnion(data, nil, nil, nil, &x.String, true, &x.StringArray, false, nil, false, nil, false, nil, true)
	if err != nil {
		return err
	}
	if object {
	}
	return nil
}

func (x *ThreadListCwdFilter) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, x.String, x.StringArray != nil, x.StringArray, false, nil, false, nil, false, nil, true)
}

type NetworkAccessUnion struct {
	Bool *bool
	Enum *NetworkAccess
}

func (x *NetworkAccessUnion) UnmarshalJSON(data []byte) error {
	x.Enum = nil
	object, err := unmarshalUnion(data, nil, nil, &x.Bool, nil, false, nil, false, nil, false, nil, true, &x.Enum, false)
	if err != nil {
		return err
	}
	if object {
	}
	return nil
}

func (x *NetworkAccessUnion) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, x.Bool, nil, false, nil, false, nil, false, nil, x.Enum != nil, x.Enum, false)
}

type CommandExecutionApprovalDecision struct {
	Enum                                            *FileChangeApprovalDecision
	PolicyAmendmentCommandExecutionApprovalDecision *PolicyAmendmentCommandExecutionApprovalDecision
}

func (x *CommandExecutionApprovalDecision) UnmarshalJSON(data []byte) error {
	x.PolicyAmendmentCommandExecutionApprovalDecision = nil
	x.Enum = nil
	var c PolicyAmendmentCommandExecutionApprovalDecision
	object, err := unmarshalUnion(data, nil, nil, nil, nil, false, nil, true, &c, false, nil, true, &x.Enum, false)
	if err != nil {
		return err
	}
	if object {
		x.PolicyAmendmentCommandExecutionApprovalDecision = &c
	}
	return nil
}

func (x *CommandExecutionApprovalDecision) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, nil, false, nil, x.PolicyAmendmentCommandExecutionApprovalDecision != nil, x.PolicyAmendmentCommandExecutionApprovalDecision, false, nil, x.Enum != nil, x.Enum, false)
}

// User's decision in response to an ExecApprovalRequest.
type ExecCommandApprovalResponseReviewDecision struct {
	Enum                 *ReviewDecision
	FluffyReviewDecision *FluffyReviewDecision
}

func (x *ExecCommandApprovalResponseReviewDecision) UnmarshalJSON(data []byte) error {
	x.FluffyReviewDecision = nil
	x.Enum = nil
	var c FluffyReviewDecision
	object, err := unmarshalUnion(data, nil, nil, nil, nil, false, nil, true, &c, false, nil, true, &x.Enum, false)
	if err != nil {
		return err
	}
	if object {
		x.FluffyReviewDecision = &c
	}
	return nil
}

func (x *ExecCommandApprovalResponseReviewDecision) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, nil, false, nil, x.FluffyReviewDecision != nil, x.FluffyReviewDecision, false, nil, x.Enum != nil, x.Enum, false)
}

type MovePath struct {
	String    *string
	TurnError *TurnError
}

func (x *MovePath) UnmarshalJSON(data []byte) error {
	x.TurnError = nil
	var c TurnError
	object, err := unmarshalUnion(data, nil, nil, nil, &x.String, false, nil, true, &c, false, nil, false, nil, true)
	if err != nil {
		return err
	}
	if object {
		x.TurnError = &c
	}
	return nil
}

func (x *MovePath) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, x.String, false, nil, x.TurnError != nil, x.TurnError, false, nil, false, nil, true)
}

type CodexErrorInfoUnion struct {
	CodexErrorInfo *CodexErrorInfo
	Enum           *CodexErrorInfoEnum
}

func (x *CodexErrorInfoUnion) UnmarshalJSON(data []byte) error {
	x.CodexErrorInfo = nil
	x.Enum = nil
	var c CodexErrorInfo
	object, err := unmarshalUnion(data, nil, nil, nil, nil, false, nil, true, &c, false, nil, true, &x.Enum, true)
	if err != nil {
		return err
	}
	if object {
		x.CodexErrorInfo = &c
	}
	return nil
}

func (x *CodexErrorInfoUnion) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, nil, false, nil, x.CodexErrorInfo != nil, x.CodexErrorInfo, false, nil, x.Enum != nil, x.Enum, true)
}

type StatusUnion struct {
	Enum         *MCPServerStartupState
	ThreadStatus *ThreadStatus
}

func (x *StatusUnion) UnmarshalJSON(data []byte) error {
	x.ThreadStatus = nil
	x.Enum = nil
	var c ThreadStatus
	object, err := unmarshalUnion(data, nil, nil, nil, nil, false, nil, true, &c, false, nil, true, &x.Enum, false)
	if err != nil {
		return err
	}
	if object {
		x.ThreadStatus = &c
	}
	return nil
}

func (x *StatusUnion) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, nil, false, nil, x.ThreadStatus != nil, x.ThreadStatus, false, nil, x.Enum != nil, x.Enum, false)
}

// Origin of the thread (CLI, VSCode, codex exec, codex app-server, etc.).
type SessionSourceUnion struct {
	Enum          *SessionSourceEnum
	SessionSource *SessionSource
}

func (x *SessionSourceUnion) UnmarshalJSON(data []byte) error {
	x.SessionSource = nil
	x.Enum = nil
	var c SessionSource
	object, err := unmarshalUnion(data, nil, nil, nil, nil, false, nil, true, &c, false, nil, true, &x.Enum, false)
	if err != nil {
		return err
	}
	if object {
		x.SessionSource = &c
	}
	return nil
}

func (x *SessionSourceUnion) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, nil, false, nil, x.SessionSource != nil, x.SessionSource, false, nil, x.Enum != nil, x.Enum, false)
}

type SubAgentSourceUnion struct {
	Enum           *SubAgentSourceEnum
	SubAgentSource *SubAgentSource
}

func (x *SubAgentSourceUnion) UnmarshalJSON(data []byte) error {
	x.SubAgentSource = nil
	x.Enum = nil
	var c SubAgentSource
	object, err := unmarshalUnion(data, nil, nil, nil, nil, false, nil, true, &c, false, nil, true, &x.Enum, false)
	if err != nil {
		return err
	}
	if object {
		x.SubAgentSource = &c
	}
	return nil
}

func (x *SubAgentSourceUnion) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, nil, false, nil, x.SubAgentSource != nil, x.SubAgentSource, false, nil, x.Enum != nil, x.Enum, false)
}

type ContentElement struct {
	String             *string
	TextUserInputClass *TextUserInputClass
}

func (x *ContentElement) UnmarshalJSON(data []byte) error {
	x.TextUserInputClass = nil
	var c TextUserInputClass
	object, err := unmarshalUnion(data, nil, nil, nil, &x.String, false, nil, true, &c, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
		x.TextUserInputClass = &c
	}
	return nil
}

func (x *ContentElement) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, x.String, false, nil, x.TextUserInputClass != nil, x.TextUserInputClass, false, nil, false, nil, false)
}

type Result struct {
	MCPToolCallResult *MCPToolCallResult
	String            *string
}

func (x *Result) UnmarshalJSON(data []byte) error {
	x.MCPToolCallResult = nil
	var c MCPToolCallResult
	object, err := unmarshalUnion(data, nil, nil, nil, &x.String, false, nil, true, &c, false, nil, false, nil, true)
	if err != nil {
		return err
	}
	if object {
		x.MCPToolCallResult = &c
	}
	return nil
}

func (x *Result) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, x.String, false, nil, x.MCPToolCallResult != nil, x.MCPToolCallResult, false, nil, false, nil, true)
}

type AskForApproval struct {
	AskForApprovalGranularAskForApproval *AskForApprovalGranularAskForApproval
	Enum                                 *ApprovalPolicyEnum
}

func (x *AskForApproval) UnmarshalJSON(data []byte) error {
	x.AskForApprovalGranularAskForApproval = nil
	x.Enum = nil
	var c AskForApprovalGranularAskForApproval
	object, err := unmarshalUnion(data, nil, nil, nil, nil, false, nil, true, &c, false, nil, true, &x.Enum, false)
	if err != nil {
		return err
	}
	if object {
		x.AskForApprovalGranularAskForApproval = &c
	}
	return nil
}

func (x *AskForApproval) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, nil, nil, nil, false, nil, x.AskForApprovalGranularAskForApproval != nil, x.AskForApprovalGranularAskForApproval, false, nil, x.Enum != nil, x.Enum, false)
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
