import { useTranslation } from 'react-i18next';
import { useNavigate } from "@tanstack/react-router";

interface SeriesCardProps {
  series: {
    name: string;
  };
}

export function SeriesCard({ series }: SeriesCardProps) {
  const navigate = useNavigate();
  const { t } = useTranslation();

  const handleViewDetails = () => {
    navigate({ to: `/series/${series.name}` });
  };

  return (
    <div
      className="glass-card flex items-center p-4 bg-white dark:bg-brand-800 border border-brand-200 dark:border-brand-700 rounded-xl shadow-sm hover:shadow-md transition-all text-left group cursor-pointer"
      onClick={handleViewDetails}
    >
      <div className="p-3 rounded-lg mr-4 bg-neutral-100 text-neutral-600 dark:bg-neutral-900/30 dark:text-neutral-400">
        <div className="text-2xl i-mdi-format-list-bulleted-type" />
      </div>
      <div className="flex-1">
        <h3 className="font-semibold text-brand-900 dark:text-white group-hover:text-neutral-600 dark:group-hover:text-neutral-400 transition-colors">
          {series.name}
        </h3>
      </div>
    </div>
  );
}