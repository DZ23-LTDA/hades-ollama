import HadesEmblem from "@/components/HadesEmblem";

interface LogoProps {
  size?: number;
  containerClassName?: string;
  showBackground?: boolean;
}

// Logo da marca Hades.
// Por padrão o emblema é integrado à página em currentColor (escuro no tema claro,
// branco no escuro) — sem caixa/tile, para não parecer um item separado.
// showBackground=true mantém o tile navy (feel de ícone de app) quando desejado.
export default function Logo({
  size = 60,
  containerClassName = "mb-8",
  showBackground = false,
}: LogoProps = {}) {
  if (showBackground) {
    const tile = Math.round(size * 1.5);
    return (
      <div className={`flex justify-center select-none ${containerClassName}`}>
        <div
          className="flex items-center justify-center rounded-2xl bg-[#0B0F17] text-white shadow-sm ring-1 ring-black/5 dark:ring-white/10"
          style={{ width: tile, height: tile }}
        >
          <HadesEmblem size={size} />
        </div>
      </div>
    );
  }
  return (
    <div className={`flex justify-center select-none ${containerClassName}`}>
      <HadesEmblem
        size={size}
        className="text-neutral-900 dark:text-white"
      />
    </div>
  );
}
