import { ChevronRight } from "lucide-react";
import { useTranslation } from "react-i18next";

interface EdgeRevealStripProps {
  onClick: () => void;
}

export function EdgeRevealStrip({ onClick }: EdgeRevealStripProps) {
  const { t } = useTranslation();

  return (
    <button
      className="fixed left-0 top-0 bottom-0 z-40 w-1 hover:w-5 overflow-hidden cursor-pointer flex items-center justify-center transition-all duration-200 bg-transparent hover:bg-muted/60 hover:backdrop-blur-sm group"
      onClick={onClick}
      title={t("panel.showSidebar")}
    >
      <ChevronRight className="h-4 w-4 text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity duration-200" />
    </button>
  );
}
