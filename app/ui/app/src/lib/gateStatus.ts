/**
 * Honest Gate & Capability Status Model
 * Prevents "fake done" and ensures every capability status reflects verified execution.
 */

export type GateStatus =
  | "PASS"
  | "FAIL"
  | "NOT_EXECUTED"
  | "NOT_PRESENT"
  | "NOT_CONFIGURED"
  | "BLOCKED_EXTERNAL"
  | "UNKNOWN";

export interface CapabilityGate {
  id: string;
  domain: string;
  capability: string;
  status: GateStatus;
  executed: boolean;
  evidenceRef?: string;
  notes?: string;
}

/**
 * Evaluates gate status ensuring the hard rule:
 * Non-executed gates are NEVER marked as PASS.
 */
export function evaluateGate(
  id: string,
  domain: string,
  capability: string,
  options: {
    executed: boolean;
    passed?: boolean;
    hasConfig?: boolean;
    externalBlocked?: boolean;
    evidenceRef?: string;
    notes?: string;
  }
): CapabilityGate {
  const {
    executed,
    passed = false,
    hasConfig = true,
    externalBlocked = false,
    evidenceRef,
    notes,
  } = options;

  let status: GateStatus;

  if (externalBlocked) {
    status = "BLOCKED_EXTERNAL";
  } else if (!hasConfig) {
    status = "NOT_CONFIGURED";
  } else if (!executed) {
    status = "NOT_EXECUTED";
  } else if (passed) {
    if (!evidenceRef || evidenceRef.trim() === "") {
      throw new Error(
        `Integrity error for gate ${id}: cannot mark PASS without verified evidenceRef`
      );
    }
    status = "PASS";
  } else {
    status = "FAIL";
  }

  return {
    id,
    domain,
    capability,
    status,
    executed,
    evidenceRef,
    notes,
  };
}
