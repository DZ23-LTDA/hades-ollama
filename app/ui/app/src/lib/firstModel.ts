// Single source of truth for the "first model" a brand-new user is offered.
// It must be small, fast to download, and reliable on modest hardware so that
// "install and use" works out of the box. Bigger/featured models stay available
// through the model picker and the backend recommendations — this is only the
// friendly default for the very first download.
//
// Used by: FirstModelCard (home zero-model card), the onboarding "run" step
// (via FIRST_MODEL_COMMAND), and the quickstart scripts.
export const RECOMMENDED_FIRST_MODEL = "qwen2.5:0.5b";

export const FIRST_MODEL_ERROR_MESSAGE =
  "Não foi possível baixar o modelo agora. Verifique sua conexão e tente novamente.";
