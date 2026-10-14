import { useState } from "react";
import { catalogAccentClasses, connectorLogoUrl } from "@/lib/connectorCatalog";

/**
 * Tile de ícone do conector. Por padrão mostra um monograma colorido (identidade
 * própria, sem depender de assets de terceiros). Quando o operador configura o
 * clientId do Brandfetch (VITE_BRANDFETCH_CLIENT_ID), tenta renderizar o logo da
 * marca pela CDN licenciada; se a imagem falhar, volta ao monograma.
 */
export function ConnectorLogo({
  id,
  name,
  category,
}: {
  id: string;
  name: string;
  category: string;
}) {
  const [failed, setFailed] = useState(false);
  const url = connectorLogoUrl(id);
  const monogram = name.slice(0, 2).toUpperCase();

  return (
    <div
      className={`flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-xl text-sm font-semibold ${catalogAccentClasses(category)}`}
    >
      {url && !failed ? (
        <img
          src={url}
          alt=""
          width={24}
          height={24}
          loading="lazy"
          className="h-6 w-6 object-contain"
          onError={() => setFailed(true)}
        />
      ) : (
        monogram
      )}
    </div>
  );
}
