import { useState, useEffect, useCallback } from 'react';
import { models, enums } from '../../../wailsjs/go/models';
import { LoadReviewsForGame, LoadDetailReview } from '../../../wailsjs/go/service/GameService';
import { OpenBrowser } from '../../../wailsjs/go/service/ImportService';
import { toast } from 'react-hot-toast';
import { formatLocalDate } from '../../utils/time';
import { useAppStore } from '../../store';
import i18next from '../../i18n/i18n';
import { BetterButton } from '../ui/BetterButton';
import { N } from '@unocss/preset-wind3/dist/rules-Dd5IWQsx.mjs';

const t = i18next.t;

interface ReviewPanelProps {
  game: models.Game;
}

const sourceTypeOptions = [
  { value: enums.SourceType.BANGUMI || "bangumi", label: "Bangumi" },
  { value: enums.SourceType.DMM || "Dmm", label: "Dmm" },
  { value: enums.SourceType.EROSCAPE || "批评空间", label: "批评空间" },
  { value: enums.SourceType.DLSITE || "Dlsite", label: "Dlsite" },
];

export function ReviewPanel({ game }: ReviewPanelProps) {
  const [gameReview, setGameReview] = useState<models.GameReview | null>(null);

  const getSource = () => {
    if (game.bangumi_id !== "") {
      return enums.SourceType.BANGUMI;
    }
    if (game.dmm_id !== "") {
      return enums.SourceType.DMM;
    }
    if (game.dlsite_id !== "") {
      return enums.SourceType.DLSITE;
    }
    if (game.eroscape_id !== "") {
      return enums.SourceType.EROSCAPE;
    }
    return null;
  }

  const [selectedSourceType, setSelectedSourceType] = useState<enums.SourceType | null>(getSource);
  const [loading, setLoading] = useState(true);
  const [loadingReviewDetail, setLoadingReviewDetail] = useState<string | null>(null);
  const [showSourceDropdown, setShowSourceDropdown] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const [hasNextPage, setHasNextPage] = useState(false);
  const config = useAppStore(state => state.config);

  console.log("review 00", game)

  const getGameId = () => {
    if (selectedSourceType == enums.SourceType.EROSCAPE) {
      return game.eroscape_id;
    } else if (selectedSourceType == enums.SourceType.YMGAL) {
      return game.ymgal_id;
    } else if (selectedSourceType == enums.SourceType.DMM) {
      return game.dmm_id;
    } else if (selectedSourceType == enums.SourceType.DLSITE) {
      return game.dlsite_id;
    } else if (selectedSourceType == enums.SourceType.BANGUMI) {
      return game.bangumi_id;
    }
    return ""
  }

  const filteredSourceTypeOptions = sourceTypeOptions.filter(option => {
    if (game.eroscape_id !== "" && option.value === enums.SourceType.EROSCAPE) {
      return true;
    } else if (game.dlsite_id !== "" && option.value === enums.SourceType.DLSITE) {
      return true;
    } else if (game.bangumi_id !== "" && option.value === enums.SourceType.BANGUMI) {
      return true;
    } else if (game.dmm_id !== "" && option.value === enums.SourceType.DMM) {
      return true;
    }
    return false;
  })

  const loadReviews = async () => {
    try {
      setLoading(true);
      var gameId = getGameId(); 
      if (gameId == "" || selectedSourceType == null) {
        return;
      }

      const reviewData = await LoadReviewsForGame(gameId, selectedSourceType, currentPage);
      console.log("reviewData:", reviewData)
      const hasNext = reviewData.has_next;
      setHasNextPage(hasNext);
      if (reviewData && reviewData.reviews && reviewData.reviews.length > 0) {
        setGameReview(reviewData);
      } else {
        setGameReview(null);
      }
    } catch (error) {
      console.error('Failed to load reviews:', error);
      toast.error(t('reviews.loadFailed') || '加载评论失败');
      setGameReview(null);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    setSelectedSourceType(getSource);
    setCurrentPage(1);
    console.log("game:", game);
    // OpenBrowser(`https://bgm.tv/subject/${game.bangumi_id}/comments`)
  }, [game]);

  useEffect(() => {
    if (selectedSourceType == enums.SourceType.EROSCAPE || selectedSourceType == enums.SourceType.DLSITE || selectedSourceType == enums.SourceType.DMM) {
      loadReviews();
    } else if (selectedSourceType == enums.SourceType.BANGUMI) {
      setLoading(false)
      // window.open(`https://bgm.tv/subject/${game.bangumi_id}/comments`)
    }
    console.log("selectedSourceType:", selectedSourceType);
    
  }, [selectedSourceType, currentPage]);

  const handleSourceTypeChange = (sourceType: enums.SourceType) => {
    setSelectedSourceType(sourceType);
    setCurrentPage(1);
    setShowSourceDropdown(false);
  };

  console.log("review 01")

  const handleLoadMore = async (review: models.Review, sourceId: string) => {
    if (sourceId == "" || selectedSourceType == null) return;
    try {
      setLoadingReviewDetail(review.id);
      const detailReview = await LoadDetailReview(review, sourceId, selectedSourceType);
      review.content = detailReview.content;
      review.title = detailReview.title;
      review.link = ""
      
      if (detailReview && gameReview) {
        setGameReview(prev => {
          if (!prev) return prev;
          const updatedReviews = prev.reviews.map(r => 
            r.id === review.id ? new models.Review(review) : r
          );
          return new models.GameReview({
            ...prev,
            reviews: updatedReviews
          });
        });
      }
    } catch (error) {
      console.error('Failed to load review detail:', error);
      toast.error(t('reviews.loadDetailFailed') || '加载评论详情失败');
    } finally {
      setLoadingReviewDetail(null);
    }
  };

  const getSourceTypeLabel = (sourceType: enums.SourceType) => {
    const option = filteredSourceTypeOptions.find(opt => opt.value === sourceType);
    return option?.label || sourceType;
  };

  const getPageNumbers = () => {
    const pages: (number | string)[] = [];
    
    if (currentPage <= 4) {
      for (let i = 1; i <= currentPage; i++) {
        pages.push(i);
      }
      // pages.push('...', 5);
    } else {
      pages.push(1, 2, '...', currentPage - 1, currentPage);
    }
    
    return pages;
  };

  const handlePrevPage = () => {
    if (currentPage > 1) {
      setCurrentPage(prev => prev - 1);
    }
  };

  const handleNextPage = () => {
    if (hasNextPage) {
      setCurrentPage(prev => prev + 1);
    }
  };

  const handlePageClick = (page: number | string) => {
    if (typeof page === 'number') {
      setCurrentPage(page);
    }
  };

  const getAverageRating = () => {
    if (!gameReview) return ["", "100"]
    try {
      var tp = 5;
      var stp = 0;
      var sap = 0;
      for (var i = 0; i < (gameReview?.reviews?.length ?? 0); i++) {
        var review = gameReview.reviews[i];        
        
        tp = Number(review.total_points);
        if (isNaN(tp)) tp = 5;
        var t = Number(review.total_points);
        if (isNaN(t)) t = tp;
        stp += t;
        var a = Number(review.points);
        if (isNaN(a)) a = 0;
        sap += a;
        console.log("point:" + a + " total:" + t);
      }
      if (stp == 0) {
        console.log("stp == 0")
        return ["", "100"]
      }
      console.log("stp: " + stp + " sap: " + sap)
      return [(1.00 * sap / stp * tp).toFixed(4), tp.toString()]
    } catch (error) {
      console.log("getAverageRating err ", error)
      return ["", "100"]
    }
    
  };

  const renderPagination = () => {
    const pageNumbers = getPageNumbers();
    
    return (
      <div className="flex items-center justify-center gap-1">
        <button
          onClick={handlePrevPage}
          disabled={currentPage === 1 || loading}
          className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
            currentPage === 1
              ? "bg-gray-100 text-gray-400 cursor-not-allowed dark:bg-gray-700 dark:text-gray-500"
              : "bg-brand-100 text-brand-700 hover:bg-brand-200 dark:bg-brand-800 dark:text-brand-300 dark:hover:bg-brand-700"
          }`}
        >
          {t('reviews.previous') || '上一页'}
        </button>
        
        {pageNumbers.map((page, index) => (
          typeof page === 'number' ? (
            <button
              key={index}
              onClick={() => handlePageClick(page)}
              disabled={loading}
              className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                page === currentPage
                  ? "bg-brand-600 text-white dark:bg-brand-500"
                  : "bg-brand-100 text-brand-700 hover:bg-brand-200 dark:bg-brand-800 dark:text-brand-300 dark:hover:bg-brand-700"
              }`}
            >
              {page}
            </button>
          ) : (
            <span key={index} className="px-2 text-brand-500 dark:text-brand-400">...</span>
          )
        ))}
        
        <button
          onClick={handleNextPage}
          disabled={!hasNextPage || loading}
          className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
            !hasNextPage
              ? "bg-gray-100 text-gray-400 cursor-not-allowed dark:bg-gray-700 dark:text-gray-500"
              : "bg-brand-100 text-brand-700 hover:bg-brand-200 dark:bg-brand-800 dark:text-brand-300 dark:hover:bg-brand-700"
          }`}
        >
          {t('reviews.next') || '下一页'}
        </button>
      </div>
    );
  };

  if (selectedSourceType == null) {
    return null;
  }

  if (loading && !gameReview) {
    return (
      <div className="review-panel flex items-center justify-center h-64">
        <div className="text-brand-500 dark:text-brand-400">{t('common.loading')}</div>
      </div>
    );
  }

  return (
    <div className="review-panel space-y-6">
      <div className="flex justify-between items-start">
        <div className="relative">
          <div className="text-sm font-medium text-brand-700 dark:text-brand-300 mb-2">{t('reviews.dataSource') || '数据源'}</div>
          <button
            onClick={() => setShowSourceDropdown(!showSourceDropdown)}
            className="flex items-center gap-2 px-4 py-2 bg-brand-100 hover:bg-brand-200 dark:bg-brand-800 dark:hover:bg-brand-700 text-brand-800 dark:text-brand-200 rounded-lg transition-colors min-w-[140px]"
          >
            <span>{getSourceTypeLabel(selectedSourceType)}</span>
            <span className="i-mdi-chevron-down text-sm ml-auto" />
          </button>
          
          {showSourceDropdown && (
            <div className="absolute top-full left-0 mt-1 w-full bg-white dark:bg-brand-800 rounded-lg shadow-lg border border-brand-200 dark:border-brand-700 z-10">
              {filteredSourceTypeOptions.map(option => (
                <button
                  key={option.value}
                  onClick={() => handleSourceTypeChange(option.value)}
                  className={`w-full text-left px-4 py-2 hover:bg-brand-100 dark:hover:bg-brand-700 transition-colors first:rounded-t-lg last:rounded-b-lg ${
                    selectedSourceType === option.value ? "bg-brand-200 dark:bg-brand-700 text-brand-800 dark:text-brand-200" : "text-brand-700 dark:text-brand-300"
                  }`}
                >
                  {option.label}
                </button>
              ))}
            </div>
          )}
        </div>

        {gameReview && (
          <div className="flex items-center gap-4">
            <div className="text-center">
              <div className="text-sm text-brand-600 dark:text-brand-400">{t('reviews.overallScore') || '综合评分'}</div>
              <div className="text-3xl font-bold text-brand-700 dark:text-brand-300">
                {getAverageRating()[0]}
                <span className="text-lg text-brand-500 dark:text-brand-400">/{getAverageRating()[1] || '100'}</span>
              </div>
            </div>
          </div>
        )}
      </div>

      {(selectedSourceType === enums.SourceType.BANGUMI) && ( 
        <BetterButton onClick={() => { 
          if (selectedSourceType == enums.SourceType.BANGUMI) { 
            OpenBrowser(`https://bgm.tv/subject/${game.bangumi_id}/comments`)
          }
        }}>
          {`前往${selectedSourceType}评论`}

          </BetterButton>
      )}

      {!gameReview || gameReview.reviews.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-12 text-brand-500 dark:text-brand-400">
          <div className="i-mdi-message-text-outline text-4xl mb-2" />
          <p>{t('reviews.noData') || '暂无评论数据'}</p>
        </div>
      ) : (
        <>
          <div className="flex justify-center">
            {renderPagination()}
          </div>

          <div className="space-y-4">
            {gameReview.reviews.map((review, index) => (
              <div 
                key={review.id || index}
                className="bg-brand-50 dark:bg-brand-900/30 rounded-lg p-4 border border-brand-200 dark:border-brand-700"
              >
                <div className="flex justify-between items-start mb-2">
                  <div className="flex-1">
                    {review.title && (
                      <h4 className="font-semibold text-brand-900 dark:text-white text-lg">
                        {review.title}
                      </h4>
                    )}
                    <div className="flex items-center gap-2 text-sm text-brand-600 dark:text-brand-400">
                      {review.reviewer && <span>{review.reviewer}</span>}
                      {review.date && (
                        <>
                          <span>•</span>
                          <span>{formatLocalDate(review.date, config?.time_zone)}</span>
                        </>
                      )}
                    </div>
                  </div>
                  {review.points && (
                    <div className="flex items-center gap-1 bg-brand-200 dark:bg-brand-700 px-3 py-1 rounded-full">
                      <span className="i-mdi-star text-yellow-500" />
                      <span className="font-semibold text-brand-800 dark:text-brand-200">
                        {review.points}
                        <span className="text-sm text-brand-600 dark:text-brand-400">/{review.total_points || '100'}</span>
                      </span>
                    </div>
                  )}
                </div>

                {review.content && (
                  <div className="mt-3 text-brand-700 dark:text-brand-300 whitespace-pre-wrap leading-relaxed">
                    {review.content}
                  </div>
                )}

                {review.id && review.link && (
                  <div className="mt-3 flex justify-end">
                    {loadingReviewDetail === review.id ? (
                      <button
                        disabled
                        className="flex items-center gap-2 px-4 py-2 bg-brand-200 dark:bg-brand-700 text-brand-800 dark:text-brand-200 rounded-lg cursor-wait"
                      >
                        <span className="i-mdi-loading animate-spin" />
                        {t('common.loading')}
                      </button>
                    ) : (
                      <button
                        onClick={() => handleLoadMore(review, getGameId())}
                        className="flex items-center gap-2 px-4 py-2 bg-brand-600 hover:bg-brand-700 dark:bg-brand-500 dark:hover:bg-brand-600 text-white rounded-lg transition-colors"
                      >
                        {t('reviews.viewMore') || '看更多'}
                        <span className="i-mdi-arrow-right text-sm" />
                      </button>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>

          <div className="flex justify-center">
            {renderPagination()}
          </div>
        </>
      )}
    </div>
  );
}
