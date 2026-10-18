package main

import (
	"context"
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/ollama/ollama/cmd"
)

func main() {
	err := cmd.NewCLI().ExecuteContext(context.Background())
	if err == nil {
		return
	}
	// FASE 8 do adendo V3: "codigos de saida documentados". Um comando pode declarar o
	// seu proprio codigo implementando ExitCode(). A interface e' ADITIVA e OPCIONAL:
	// nenhum comando existente a implementa, por isso todos continuam a sair por
	// `cobra.CheckErr` (exit 1). *Um contrato novo que muda o comportamento dos
	// consumidores antigos nao e' um contrato: e' uma quebra.*
	var exitCoder interface{ ExitCode() int }
	if errors.As(err, &exitCoder) {
		if code := exitCoder.ExitCode(); code != 0 {
			cobra.CheckErr(err)
		}
		os.Exit(exitCoder.ExitCode())
	}
	cobra.CheckErr(err)
}
