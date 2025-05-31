package main

import (
	"log"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// --- Scoring Logic (unchanged) ---

// WordScorerConfig holds the configuration for the word scorer.
type WordScorerConfig struct {
	MinScore                  int
	MaxScore                  int
	ThresholdWordsForMaxScore int
}

// WordScorer calculates scores based on word count.
type WordScorer struct {
	config WordScorerConfig
}

// NewWordScorer creates a new WordScorer with the given configuration.
func NewWordScorer(config WordScorerConfig) (*WordScorer, error) {
	if config.MinScore < 0 {
		return nil, log.Output(1, "MinScore cannot be negative")
	}
	if config.MaxScore < config.MinScore {
		return nil, log.Output(1, "MaxScore cannot be less than MinScore")
	}
	if config.ThresholdWordsForMaxScore <= 0 {
		return nil, log.Output(1, "ThresholdWordsForMaxScore must be positive")
	}
	return &WordScorer{config: config}, nil
}

// CountWords splits the text by whitespace and returns the number of tokens.
func (s *WordScorer) CountWords(text string) int {
	return len(strings.Fields(text))
}

// CalculateScore calculates the score for the given text.
func (s *WordScorer) CalculateScore(text string) int {
	wordCount := s.CountWords(text)

	if wordCount == 0 {
		return s.config.MinScore
	}

	if wordCount >= s.config.ThresholdWordsForMaxScore {
		return s.config.MaxScore
	}

	scoreRange := float64(s.config.MaxScore - s.config.MinScore)
	progress := float64(wordCount) / float64(s.config.ThresholdWordsForMaxScore)
	calculatedScore := float64(s.config.MinScore) + (progress * scoreRange)

	finalScore := int(math.Round(calculatedScore))

	if finalScore < s.config.MinScore {
		return s.config.MinScore
	}
	if finalScore > s.config.MaxScore {
		return s.config.MaxScore
	}
	return finalScore
}

// --- API Specific Structs (unchanged from previous "no ID" version) ---

// ScoreRequest defines the structure of the JSON request body.
type ScoreRequest struct {
	TextToScore string `json:"documentText" binding:"required"`
}

// ScoreResponse defines the structure of the JSON response.
type ScoreResponse struct {
	OriginalText string `json:"originalText,omitempty"`
	WordCount    int    `json:"wordCount"`
	Score        int    `json:"score"`
	Message      string `json:"message,omitempty"`
}

// --- Handler ---

// ScoreHandler holds dependencies for the scoring API handlers.
type ScoreHandler struct {
	scorer *WordScorer
}

// NewScoreHandler creates a new ScoreHandler with its dependencies.
func NewScoreHandler(scorer *WordScorer) *ScoreHandler {
	return &ScoreHandler{
		scorer: scorer,
	}
}

// PostScore is the Gin handler function for POST /api/score.
// It's now a method of the ScoreHandler struct.
func (h *ScoreHandler) PostScore(c *gin.Context) {
	var requestBody ScoreRequest

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, ScoreResponse{
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Ensure the scorer is available (it should be, as it's part of the handler struct)
	if h.scorer == nil {
		log.Println("Error: Scorer not initialized in handler!")
		c.JSON(http.StatusInternalServerError, ScoreResponse{
			Message: "Server configuration error: scorer not available.",
		})
		return
	}

	text := requestBody.TextToScore
	wordCount := h.scorer.CountWords(text)
	score := h.scorer.CalculateScore(text)

	response := ScoreResponse{
		WordCount: wordCount,
		Score:     score,
		Message:   "Scoring successful",
	}

	// Optionally include original text
	// if len(text) < 200 {
	//  response.OriginalText = text
	// }

	c.JSON(http.StatusOK, response)
}

func main() {
	// 1. Initialize the Scorer
	scorerConfig := WordScorerConfig{
		MinScore:                  20,
		MaxScore:                  40,
		ThresholdWordsForMaxScore: 50,
	}
	appScorer, err := NewWordScorer(scorerConfig)
	if err != nil {
		log.Fatalf("Failed to initialize scorer: %v", err)
	}
	log.Printf("Scorer initialized: MinScore=%d, MaxScore=%d, Threshold=%d words",
		scorerConfig.MinScore, scorerConfig.MaxScore, scorerConfig.ThresholdWordsForMaxScore)

	// 2. Initialize the Handler
	scoreAPIHandler := NewScoreHandler(appScorer)

	// 3. Initialize Gin router
	router := gin.Default()

	// 4. Define API routes under an /api group
	apiRoutes := router.Group("/api")
	{
		apiRoutes.POST("/score", scoreAPIHandler.PostScore) // Use the method from the handler instance
		// You could add more routes here, e.g., apiRoutes.GET("/config", scoreAPIHandler.GetConfig)
	}

	// 5. Start the server
	port := "8080"
	log.Printf("Starting server on port %s, API available at /api/score", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}