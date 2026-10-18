[Types]
; Gerado a partir de internal/agent/install_profile.go. NAO EDITAR A MAO.
Name: "basic"; Description: "Basico - HADES Core, Mission Control e diagnostico"
Name: "recommended"; Description: "Recomendado - Core + IA local, navegador controlado e plugins"
Name: "developer-complete"; Description: "Desenvolvimento - Core + toolchains de codigo e release"
Name: "mobile"; Description: "Movel - Core + Android, Expo e Flutter"
Name: "media-and-voice"; Description: "Media e voz - Core + media, voz e OCR"
Name: "browser-and-computer-use"; Description: "Navegador e Computer Use - Core + controlo do computador"
Name: "containers-and-cloud"; Description: "Containers e nuvem - Core + Docker e CLIs de cloud"
Name: "custom"; Description: "Personalizado - escolha componente a componente"; Flags: iscustom
Name: "full-offline"; Description: "Offline completo - Core + IA local e toolchains sem rede"

[Components]
; Gerado a partir de internal/agent/install_profile.go. NAO EDITAR A MAO.
; `hades-core` fica de fora: e' obrigatorio e vai no [Files] sem condicao.
Name: "local-ai"; Description: "IA local (Ollama)"; Types: full-offline recommended; Flags: unchecked
Name: "browser-automation"; Description: "Navegador controlado"; Types: browser-and-computer-use full-offline recommended
Name: "computer-use"; Description: "Computer Use"; Types: browser-and-computer-use; Flags: unchecked
Name: "media"; Description: "Media (FFmpeg/FFprobe)"; Types: media-and-voice
Name: "voice"; Description: "Voz (STT/TTS)"; Types: media-and-voice; Flags: unchecked
Name: "vision-ocr"; Description: "Visao e OCR"; Types: media-and-voice
Name: "developer-js"; Description: "Desenvolvimento JS"; Types: developer-complete full-offline mobile
Name: "developer-python"; Description: "Desenvolvimento Python"; Types: developer-complete full-offline
Name: "developer-go-rust"; Description: "Desenvolvimento Go e Rust"; Types: developer-complete
Name: "developer-dotnet-windows"; Description: ".NET e Windows SDK"; Types: developer-complete; Flags: unchecked
Name: "mobile-android"; Description: "Android SDK e ADB"; Types: developer-complete mobile; Flags: unchecked
Name: "mobile-expo"; Description: "Expo e EAS"; Types: mobile; Flags: unchecked
Name: "mobile-flutter"; Description: "Flutter e Dart"; Types: mobile; Flags: unchecked
Name: "containers"; Description: "Containers e Buildx"; Types: containers-and-cloud; Flags: unchecked
Name: "release-security"; Description: "SBOM, checksums e Cosign"; Types: containers-and-cloud developer-complete full-offline
Name: "plugins-mcp-skills"; Description: "Plugins, MCPs e skills"; Types: recommended
Name: "cloud-and-store-clis"; Description: "CLIs de cloud e lojas"; Types: containers-and-cloud; Flags: unchecked
