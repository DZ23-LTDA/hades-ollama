import AsyncStorage from "@react-native-async-storage/async-storage";
import * as Notifications from "expo-notifications";
import * as SecureStore from "expo-secure-store";
import { StatusBar } from "expo-status-bar";
import { useEffect, useMemo, useRef, useState } from "react";
import { ActivityIndicator, Alert, AppState, KeyboardAvoidingView, Platform, Pressable, SafeAreaView, ScrollView, StyleSheet, Text, TextInput, View } from "react-native";
import { buildApprovalDecisionPayload, missionStorageKey, pushStorageKey, queueStorageKey, shouldQueueOffline } from "./offlinePolicy";

Notifications.setNotificationHandler({ handleNotification: async () => ({ shouldShowAlert: true, shouldShowBanner: true, shouldShowList: true, shouldPlaySound: true, shouldSetBadge: false }) });

type Mission = { id: string; version?: number; objective: string; state: string; approvals?: Array<{ id: string; step_id: string; status: string; nonce?: string }>; last_error?: string };
type Event = { id: string; type: string; step_id?: string; created_at: string };
type Session = { access_token: string; user: { email: string }; organization: { id: string; name: string } };
type AuthSession = { authenticated: boolean; user?: { id?: string }; organization?: { id?: string; name?: string } };
type QueuedAction = { id: string; path: string; method: string; body?: string; headers?: Record<string, string>; created_at: string; attempts: number; next_attempt_at: string; idempotency_key: string; conflict?: string };

class ApiError extends Error {
  status: number;
  constructor(message: string, status: number) { super(message); this.status = status; }
}

async function request<T>(base: string, path: string, token?: string, init?: RequestInit): Promise<T> {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  if (init?.headers && !(init.headers instanceof Headers)) Object.assign(headers, init.headers);
  const response = await fetch(`${base.replace(/\/$/, "")}${path}`, { ...init, headers });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) throw new ApiError(body.error ?? response.statusText, response.status);
  return body as T;
}

export default function App() {
	const [base, setBase] = useState("http://localhost:11434");
	const [draftBase, setDraftBase] = useState(base);
	const [token, setToken] = useState("");
	const [draftToken, setDraftToken] = useState("");
	const [organizationID, setOrganizationID] = useState<string | null>(null);
	const [subjectID, setSubjectID] = useState<string | null>(null);
  const [objective, setObjective] = useState("");
  const [mission, setMission] = useState<Mission | null>(null);
  const [events, setEvents] = useState<Event[]>([]);
  const [busy, setBusy] = useState(false);
	  const [error, setError] = useState("");
	  const [online, setOnline] = useState(true);
	  const [queued, setQueued] = useState(0);
		const [conflicts, setConflicts] = useState(0);
	  const [pushRegistered, setPushRegistered] = useState(false);
	  const [approvalReasons, setApprovalReasons] = useState<Record<string, string>>({});
	  const scopeEpoch = useRef(0);

	const expireSession = async () => {
		scopeEpoch.current += 1;
		await SecureStore.deleteItemAsync("dz23.agent.token");
		setToken("");
		setDraftToken("");
		setOrganizationID(null);
		setSubjectID(null);
		setPushRegistered(false);
		setMission(null);
		setEvents([]);
		setApprovalReasons({});
		setQueued(0);
		setConflicts(0);
		setBusy(false);
	};

	  const loadQueue = async (expectedEpoch = scopeEpoch.current) => {
	    const epoch = expectedEpoch;
	    const queueKey = queueStorageKey(base, organizationID, subjectID, Boolean(token));
	    if (!queueKey) {
	      if (epoch !== scopeEpoch.current) return [];
	      setQueued(0);
	      setConflicts(0);
	      return [];
	    }
	    const raw = await AsyncStorage.getItem(queueKey);
	    if (epoch !== scopeEpoch.current) return [];
    const parsed = raw ? JSON.parse(raw) : [];
    const items: QueuedAction[] = Array.isArray(parsed) ? parsed.map((item: Partial<QueuedAction> & { token?: string }) => {
      const { token: _discardedToken, ...safe } = item;
      return { ...safe, attempts: item.attempts ?? 0, next_attempt_at: item.next_attempt_at ?? new Date(0).toISOString(), idempotency_key: item.idempotency_key ?? item.id ?? `offline_${Date.now()}` } as QueuedAction;
    }) : [];
    setQueued(items.length);
    setConflicts(items.filter((item) => item.conflict).length);
    return items;
  };

	  const enqueue = async (path: string, method: string, body?: unknown, headers?: Record<string, string>) => {
    const epoch = scopeEpoch.current;
	    if (token && (!organizationID || !subjectID)) {
	      setError("A ação offline não foi enfileirada porque a organização autenticada ainda não foi confirmada.");
	      return false;
	    }
	    const queueKey = queueStorageKey(base, organizationID, subjectID, Boolean(token));
	    if (!queueKey) return false;
	    const items = await loadQueue(epoch); if (epoch !== scopeEpoch.current) return false;
    const id = `offline_${Date.now()}_${Math.random().toString(36).slice(2)}`;
	    const idempotencyKey = `mobile_${id}`;
	    items.push({ id, path, method, body: body ? JSON.stringify(body) : undefined, headers: { ...headers, "Idempotency-Key": idempotencyKey }, created_at: new Date().toISOString(), attempts: 0, next_attempt_at: new Date().toISOString(), idempotency_key: idempotencyKey });
	    await AsyncStorage.setItem(queueKey, JSON.stringify(items)); if (epoch !== scopeEpoch.current) return false; await loadQueue(epoch);
	    return true;
	  };

	  const flushQueue = async () => {
    const epoch = scopeEpoch.current;
	    if (!organizationID || !subjectID) return;
	    const queueKey = queueStorageKey(base, organizationID, subjectID, Boolean(token));
	    if (!queueKey) return;
	    const items = await loadQueue(epoch);
	    const remaining: QueuedAction[] = [];
	    for (const item of items) {
	      if (epoch !== scopeEpoch.current) return;
      if (item.conflict || new Date(item.next_attempt_at).getTime() > Date.now()) {
        remaining.push(item);
        continue;
      }
      try {
	        await request(base, item.path, token || undefined, { method: item.method, body: item.body, headers: item.headers });
        if (epoch !== scopeEpoch.current) return;
      } catch (cause) {
        if (epoch !== scopeEpoch.current) return;
        const apiError = cause instanceof ApiError ? cause : undefined;
		        if (apiError?.status === 409) {
	          remaining.push({ ...item, conflict: "Servidor mudou a missão; revise o estado atual antes de reenviar." });
			} else if (apiError?.status === 401 || apiError?.status === 403) {
				await expireSession();
				remaining.push({ ...item, conflict: "Sessão expirada ou sem permissão; autentique novamente antes de reenviar." });
			} else if (apiError || !shouldQueueOffline(cause)) {
				remaining.push({ ...item, conflict: `Servidor respondeu HTTP ${apiError?.status ?? "inesperado"}; revise a ação antes de reenviar.` });
	        } else if (item.attempts >= 4) {
          remaining.push({ ...item, conflict: "Limite de tentativas atingido; revise a ação antes de reenviar." });
        } else {
          const attempts = item.attempts + 1;
          const delay = Math.min(60_000, 2_000 * 2 ** attempts);
          remaining.push({ ...item, attempts, next_attempt_at: new Date(Date.now() + delay).toISOString() });
        }
      }
	    }
	    if (epoch !== scopeEpoch.current) return;
	    await AsyncStorage.setItem(queueKey, JSON.stringify(remaining));
	    setQueued(remaining.length);
	    setConflicts(remaining.filter((item) => Boolean(item.conflict)).length);
	    if (remaining.length === 0) setOnline(true);
	  };

	  const hydrateAuthScope = async (server = base, bearer = token) => {
	    const epoch = scopeEpoch.current;
	    try {
	      const session = await request<AuthSession>(server, "/api/agent/v1/auth/session", bearer || undefined);
	      if (epoch !== scopeEpoch.current) return;
	      if (!session.authenticated) {
	        if (bearer) await expireSession();
	        setOrganizationID("local");
	        setSubjectID("local");
	        setOnline(true);
	        return;
	      }
	      const nextOrganization = session.organization?.id?.trim();
	      const nextSubject = session.user?.id?.trim();
	      if (!nextOrganization || !nextSubject) throw new Error("A sessão autenticada não informou organization_id e user_id.");
	      setOrganizationID(nextOrganization);
	      setSubjectID(nextSubject);
	      setOnline(true);
	    } catch (cause) {
	      if (epoch !== scopeEpoch.current) return;
	      if (cause instanceof ApiError && (cause.status === 401 || cause.status === 403)) {
	        if (bearer) {
	          await expireSession();
	          setError("Sessão expirada ou sem organização/usuário confirmados.");
	        }
	        setOrganizationID(null);
	        setSubjectID(null);
	        setOnline(true);
	      } else {
	        setOrganizationID(null);
	        setSubjectID(null);
	        setOnline(cause instanceof ApiError);
	      }
	    }
	  };

	  const refresh = async (missionId = mission?.id) => {
	    if (!missionId) return;
	    const epoch = scopeEpoch.current;
	    try {
      const [nextMission, nextEvents] = await Promise.all([
        request<Mission>(base, `/api/agent/v1/missions/${encodeURIComponent(missionId)}`, token),
	      request<{ events: Event[] }>(base, `/api/agent/v1/missions/${encodeURIComponent(missionId)}/events`, token),
	    ]);
	    if (epoch !== scopeEpoch.current) return;
	    setMission(nextMission); setEvents(nextEvents.events); setOnline(true);
	      const cacheKey = missionStorageKey(base, organizationID, subjectID, Boolean(token));
	      if (cacheKey) await AsyncStorage.setItem(cacheKey, JSON.stringify({ mission: nextMission, events: nextEvents.events }));
	    void flushQueue();
	  } catch (cause) {
	    if (epoch !== scopeEpoch.current) return;
	    if (cause instanceof ApiError && (cause.status === 401 || cause.status === 403)) { await expireSession(); setError("Sessão expirada ou sem permissão; autentique novamente."); setOnline(false); return; }
	    setOnline(false);
	      const cacheKey = missionStorageKey(base, organizationID, subjectID, Boolean(token));
	      const cached = cacheKey ? await AsyncStorage.getItem(cacheKey) : null;
      if (cached && !mission) { const value = JSON.parse(cached) as { mission: Mission; events: Event[] }; setMission(value.mission); setEvents(value.events); }
    }
  };

	  const registerPush = async (server = base, bearer = token) => {
	    if (!bearer || !server || !organizationID || !subjectID || pushRegistered) return;
	    const epoch = scopeEpoch.current;
	    try {
	      const permission = await Notifications.getPermissionsAsync();
	      if (epoch !== scopeEpoch.current) return;
	      const granted = permission.granted || (await Notifications.requestPermissionsAsync()).granted;
	      if (epoch !== scopeEpoch.current) return;
	      if (!granted) return;
	      const pushToken = (await Notifications.getExpoPushTokenAsync()).data;
	      if (epoch !== scopeEpoch.current) return;
      const platform = Platform.OS === "ios" ? "ios" : "android";
	      await request(server, "/api/agent/v1/notifications/register", bearer, { method: "POST", body: JSON.stringify({ token: pushToken, platform }) });
	      if (epoch !== scopeEpoch.current) return;
	      const pushKeyForScope = pushStorageKey(server, organizationID, subjectID);
	      if (pushKeyForScope) await AsyncStorage.setItem(pushKeyForScope, "1");
      setPushRegistered(true);
    } catch { /* Push is optional; offline mission use must continue. */ }
  };

	  useEffect(() => {
	    void AsyncStorage.getItem("dz23.agent.base").then((value) => { if (value) { setBase(value); setDraftBase(value); } });
	    void SecureStore.getItemAsync("dz23.agent.token").then((value) => { if (value) { setToken(value); setDraftToken(value); } });
	  }, []);
	  useEffect(() => {
	    void hydrateAuthScope(base, token);
	    const pushKeyForScope = pushStorageKey(base, organizationID, subjectID);
	    if (pushKeyForScope) void AsyncStorage.getItem(pushKeyForScope).then((value) => setPushRegistered(value === "1"));
	    void loadQueue();
	    const cacheKey = missionStorageKey(base, organizationID, subjectID, Boolean(token));
	    if (cacheKey) void AsyncStorage.getItem(cacheKey).then((raw) => { if (raw) { const value = JSON.parse(raw) as { mission: Mission; events: Event[] }; setMission(value.mission); setEvents(value.events); } });
	  }, [base, organizationID, subjectID, token]);
	const pollInFlight = useRef(false);
	useEffect(() => { const timer = setInterval(async () => { if (pollInFlight.current) return; pollInFlight.current = true; try { await refresh(); } finally { pollInFlight.current = false; } }, 3000); return () => clearInterval(timer); }, [mission?.id, base, token, organizationID, subjectID]);
	  useEffect(() => { void registerPush(); }, [base, token, organizationID, subjectID, pushRegistered]);
	  const flushQueueRef = useRef(flushQueue);
	  flushQueueRef.current = flushQueue;
	  useEffect(() => {
	    const timer = setInterval(() => void flushQueueRef.current(), 5000);
	    const subscription = AppState.addEventListener("change", (state) => {
	      if (state === "active") void flushQueueRef.current();
	    });
	    return () => {
	      clearInterval(timer);
	      subscription.remove();
	    };
	  }, []);

  const approvals = useMemo(() => mission?.approvals?.filter((approval) => approval.status === "PENDING") ?? [], [mission]);
  const mutationHeaders = () => mission?.version ? { "If-Match": String(mission.version) } : undefined;
  const perform = async (path: string, method: string, body: unknown, fallback: string) => {
	    const headers = mutationHeaders();
	    const epoch = scopeEpoch.current;
	    try { await request(base, path, token, { method, body: JSON.stringify(body), headers }); if (epoch !== scopeEpoch.current) return false; setOnline(true); await flushQueue(); return true; }
		catch (cause) {
			if (epoch !== scopeEpoch.current) return false;
			const apiError = cause instanceof ApiError ? cause : undefined;
				if (apiError?.status === 401 || apiError?.status === 403) { await expireSession(); setOnline(false); setError("Sessão expirada ou sem permissão; autentique novamente."); return false; }
				if (apiError?.status === 409) { setError(`${fallback}: o servidor detectou conflito. Atualizando a missão para revisão.`); await refresh(); return false; }
				if (apiError || !shouldQueueOffline(cause)) { setOnline(true); setError(`${fallback}: o servidor respondeu HTTP ${apiError?.status ?? "inesperado"}; a ação não foi enfileirada.`); return false; }
	      const queuedOffline = await enqueue(path, method, body, headers); setOnline(false); setError(queuedOffline ? `${fallback}. A ação foi salva e será sincronizada quando houver conexão.` : `${fallback}. Ação não salva: confirme a organização autenticada antes de operar offline.`); return false;
    }
  };

	  const create = async () => {
	    if (!objective.trim()) return;
	    const epoch = scopeEpoch.current;
	    setBusy(true); setError("");
	    const payload = { objective, auto_run: false };
	    try { const created = await request<Mission>(base, "/api/agent/v1/missions", token, { method: "POST", body: JSON.stringify(payload) }); if (epoch !== scopeEpoch.current) return; setMission(created); setOnline(true); await refresh(created.id); if (epoch !== scopeEpoch.current) return; }
					catch (cause) { if (epoch !== scopeEpoch.current) return; if (cause instanceof ApiError && (cause.status === 401 || cause.status === 403)) { await expireSession(); setOnline(false); setError("Sessão expirada ou sem permissão; autentique novamente."); setBusy(false); return; } if (cause instanceof ApiError || !shouldQueueOffline(cause)) { setOnline(true); setError(`O servidor respondeu HTTP ${cause instanceof ApiError ? cause.status : "inesperado"}; a missão não foi enfileirada.`); setBusy(false); return; } const queuedOffline = await enqueue("/api/agent/v1/missions", "POST", payload); if (epoch !== scopeEpoch.current) return; setOnline(false); setError(queuedOffline ? "Servidor indisponível. A missão foi salva e será sincronizada quando houver conexão." : "Servidor indisponível. A missão não foi salva porque a organização autenticada ainda não foi confirmada."); }
	    if (epoch !== scopeEpoch.current) return;
	    setObjective(""); setBusy(false);
	  };
	const decide = async (approvalId: string, nonce: string | undefined, approved: boolean) => { const reason = (approvalReasons[approvalId] ?? "").trim(); if (!mission || busy) return; if (!reason) { setError("Informe o motivo antes de decidir este approval."); return; } setBusy(true); setError(""); const payload = buildApprovalDecisionPayload(approved, nonce, reason); const succeeded = await perform(`/api/agent/v1/missions/${mission.id}/approvals/${approvalId}`, "POST", payload, "Falha ao decidir approval"); if (succeeded) setApprovalReasons((current) => ({ ...current, [approvalId]: "" })); await refresh(); setBusy(false); };
	const run = async () => { if (!mission) return; setBusy(true); setError(""); await perform(`/api/agent/v1/missions/${mission.id}/run`, "POST", {}, "Falha ao executar"); await refresh(); setBusy(false); };
	  const saveSession = async () => { const nextBase = draftBase.trim(); const nextToken = draftToken.trim(); if (!nextBase) return; scopeEpoch.current += 1; setBusy(false); await AsyncStorage.setItem("dz23.agent.base", nextBase); if (nextToken) await SecureStore.setItemAsync("dz23.agent.token", nextToken); else await SecureStore.deleteItemAsync("dz23.agent.token"); setBase(nextBase); setToken(nextToken); setOrganizationID(null); setSubjectID(null); setObjective(""); setMission(null); setEvents([]); setApprovalReasons({}); setQueued(0); setConflicts(0); setError(""); setPushRegistered(false); };
	const clearSession = async () => {
		const pending = await loadQueue();
		if (pending.length > 0 || mission || events.length > 0) {
			const discard = await new Promise<boolean>((resolve) => Alert.alert("Apagar sessão local?", "Isso remove a missão em cache, eventos e ações offline pendentes.", [{ text: "Cancelar", style: "cancel", onPress: () => resolve(false) }, { text: "Sair e apagar", style: "destructive", onPress: () => resolve(true) }], { cancelable: true, onDismiss: () => resolve(false) }));
			if (!discard) return;
		}
			const storageKeys = [queueStorageKey(base, organizationID, subjectID, Boolean(token)), missionStorageKey(base, organizationID, subjectID, Boolean(token)), pushStorageKey(base, organizationID, subjectID)].filter((key): key is string => Boolean(key));
			await expireSession();
			await AsyncStorage.multiRemove(storageKeys);
		setMission(null); setEvents([]); setQueued(0); setConflicts(0); setApprovalReasons({});
	};
	const discardConflicts = async () => { const items = await loadQueue(); const queueKey = queueStorageKey(base, organizationID, subjectID, Boolean(token)); if (queueKey) await AsyncStorage.setItem(queueKey, JSON.stringify(items.filter((item) => !item.conflict))); await loadQueue(); };

	  return <SafeAreaView style={styles.safe}><StatusBar style="auto" /><KeyboardAvoidingView style={styles.keyboard} behavior={Platform.OS === "ios" ? "padding" : undefined}><ScrollView contentContainerStyle={styles.container} keyboardShouldPersistTaps="handled">
    <Text style={styles.eyebrow}>DZ23 AGENTIC</Text><Text style={styles.title}>Mission mobile</Text><Text style={styles.subtitle}>Acompanhe, aprove e execute missões, com outbox offline, reconciliação de conflitos e notificações push.</Text>
    <View style={styles.sync}><View style={[styles.syncDot, { backgroundColor: online ? "#059669" : "#d97706" }]} /><Text style={styles.syncText}>{online ? "Online" : "Offline — cache local ativo"}{queued ? ` · ${queued} ação(ões) pendente(s)` : ""}{pushRegistered ? " · push ativo" : ""}</Text></View>
    {conflicts ? <View style={styles.conflict}><Text style={styles.errorText}>{conflicts} ação(ões) em conflito aguardam revisão.</Text><Pressable onPress={() => void discardConflicts()}><Text style={styles.link}>Descartar conflitos</Text></Pressable></View> : null}
    <View style={styles.card}><Text style={styles.label}>Servidor</Text><TextInput value={draftBase} onChangeText={setDraftBase} autoCapitalize="none" autoCorrect={false} style={styles.input} /><Text style={styles.label}>Token Bearer (armazenado no SecureStore)</Text><TextInput value={draftToken} onChangeText={setDraftToken} autoCapitalize="none" autoCorrect={false} secureTextEntry style={styles.input} /><View style={styles.row}><Pressable onPress={() => void saveSession()} style={[styles.secondary, { flex: 1 }]}><Text style={styles.secondaryText}>Salvar sessão</Text></Pressable><Pressable onPress={() => void clearSession()} style={styles.secondary}><Text style={styles.secondaryText}>Sair</Text></Pressable></View></View>
    <View style={styles.card}><Text style={styles.label}>Novo objetivo</Text><TextInput value={objective} onChangeText={setObjective} multiline placeholder="Ex.: verificar os testes do projeto" style={[styles.input, styles.multiline]} /><Pressable disabled={busy || !objective.trim()} onPress={() => void create()} style={[styles.primary, (!objective.trim() || busy) && styles.disabled]}>{busy ? <ActivityIndicator color="#fff" /> : <Text style={styles.primaryText}>Criar missão</Text>}</Pressable></View>
    {error ? <Text style={styles.error}>{error}</Text> : null}
	    {mission ? <View style={styles.card}><View style={styles.row}><View style={{ flex: 1 }}><Text style={styles.muted}>{mission.id} · v{mission.version ?? "?"}</Text><Text style={styles.mission}>{mission.objective}</Text></View><Text style={styles.status}>{mission.state}</Text></View><Pressable onPress={() => void run()} disabled={busy || approvals.length > 0 || mission.state === "COMPLETED"} accessibilityRole="button" accessibilityLabel="Executar missão" style={[styles.secondary, (busy || approvals.length > 0) && styles.disabled]}><Text style={styles.secondaryText}>Executar missão</Text></Pressable>{approvals.map((approval) => <View key={approval.id} style={styles.approval}><Text style={styles.label}>Approval: {approval.step_id}</Text><TextInput value={approvalReasons[approval.id] ?? ""} onChangeText={(value) => setApprovalReasons((current) => ({ ...current, [approval.id]: value }))} placeholder="Por que aprovar ou rejeitar?" accessibilityLabel={`Motivo do approval ${approval.step_id}`} multiline style={[styles.input, styles.approvalReason]} /><View style={styles.row}><Pressable onPress={() => void decide(approval.id, approval.nonce, true)} disabled={busy || !(approvalReasons[approval.id] ?? "").trim()} accessibilityRole="button" accessibilityLabel={`Aprovar ${approval.step_id}`} style={[styles.approve, (busy || !(approvalReasons[approval.id] ?? "").trim()) && styles.disabled]}><Text style={styles.primaryText}>Aprovar</Text></Pressable><Pressable onPress={() => void decide(approval.id, approval.nonce, false)} disabled={busy || !(approvalReasons[approval.id] ?? "").trim()} accessibilityRole="button" accessibilityLabel={`Rejeitar ${approval.step_id}`} style={[styles.reject, (busy || !(approvalReasons[approval.id] ?? "").trim()) && styles.disabled]}><Text style={styles.primaryText}>Rejeitar</Text></Pressable></View></View>)}<Text style={styles.label}>Timeline</Text>{events.map((event) => <View key={event.id} style={styles.event}><View style={styles.dot} /><View><Text style={styles.eventType}>{event.type}</Text><Text style={styles.muted}>{event.step_id ?? "mission"} · {new Date(event.created_at).toLocaleString()}</Text></View></View>)}</View> : null}
	  </ScrollView></KeyboardAvoidingView></SafeAreaView>;
}

const styles = StyleSheet.create({ safe: { flex: 1, backgroundColor: "#f7f7f5" }, keyboard: { flex: 1 }, container: { padding: 20, gap: 16 }, eyebrow: { color: "#737373", fontSize: 12, letterSpacing: 2, fontWeight: "700" }, title: { color: "#171717", fontSize: 30, fontWeight: "700" }, subtitle: { color: "#525252", fontSize: 15, lineHeight: 22 }, sync: { flexDirection: "row", alignItems: "center", gap: 8 }, syncDot: { width: 9, height: 9, borderRadius: 5 }, syncText: { color: "#525252", fontSize: 13 }, card: { backgroundColor: "#fff", borderRadius: 18, padding: 16, gap: 12, shadowColor: "#000", shadowOpacity: 0.06, shadowRadius: 12, elevation: 2 }, label: { color: "#404040", fontSize: 13, fontWeight: "600" }, muted: { color: "#737373", fontSize: 12 }, input: { borderColor: "#d4d4d4", borderWidth: 1, borderRadius: 12, padding: 12, color: "#171717", fontSize: 15 }, multiline: { minHeight: 90, textAlignVertical: "top" }, approvalReason: { minHeight: 64, textAlignVertical: "top" }, primary: { backgroundColor: "#171717", borderRadius: 12, minHeight: 44, alignItems: "center", justifyContent: "center", paddingHorizontal: 16 }, primaryText: { color: "#fff", fontSize: 14, fontWeight: "700" }, secondary: { borderColor: "#d4d4d4", borderWidth: 1, borderRadius: 12, minHeight: 42, alignItems: "center", justifyContent: "center", paddingHorizontal: 14 }, secondaryText: { color: "#262626", fontSize: 14, fontWeight: "600" }, disabled: { opacity: 0.4 }, error: { color: "#b91c1c", backgroundColor: "#fee2e2", borderRadius: 12, padding: 12 }, errorText: { color: "#92400e", flex: 1 }, conflict: { backgroundColor: "#fffbeb", borderColor: "#fcd34d", borderWidth: 1, borderRadius: 12, padding: 12, gap: 8 }, link: { color: "#92400e", fontWeight: "700" }, row: { flexDirection: "row", alignItems: "center", gap: 10 }, mission: { color: "#171717", fontSize: 16, fontWeight: "600", marginTop: 4 }, status: { color: "#525252", backgroundColor: "#f5f5f5", borderRadius: 20, paddingVertical: 6, paddingHorizontal: 10, fontSize: 12 }, approval: { backgroundColor: "#fffbeb", borderColor: "#fde68a", borderWidth: 1, borderRadius: 12, padding: 12, gap: 10 }, approve: { backgroundColor: "#059669", borderRadius: 10, paddingVertical: 10, paddingHorizontal: 14 }, reject: { backgroundColor: "#dc2626", borderRadius: 10, paddingVertical: 10, paddingHorizontal: 14 }, event: { flexDirection: "row", gap: 10, alignItems: "flex-start", paddingVertical: 8 }, dot: { width: 8, height: 8, borderRadius: 4, backgroundColor: "#737373", marginTop: 5 }, eventType: { color: "#262626", fontSize: 14, fontWeight: "600" } });
