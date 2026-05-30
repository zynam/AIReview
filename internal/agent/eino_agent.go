package agent

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"aireview/internal/agent/nodes"
	"aireview/internal/review"

	"github.com/cloudwego/eino/compose"
)

const (
	nodeContextBuild     = "context_build"
	nodeContextRetrieve  = "context_retrieve"
	nodeContextCompress  = "context_compress"
	nodePromptBuild      = "prompt_build"
	nodeChatModel        = "chat_model"
	nodeJSONParse        = "json_parse"
	nodeFindingValidate  = "finding_validate"
	nodeMerge            = "merge"
	defaultGraphRunSteps = 32
)

type EinoReviewAgent struct {
	httpClient *http.Client
}

type Option func(*EinoReviewAgent)

func NewEinoReviewAgent(opts ...Option) *EinoReviewAgent {
	agent := &EinoReviewAgent{}
	for _, opt := range opts {
		opt(agent)
	}
	return agent
}

func WithHTTPClient(client *http.Client) Option {
	return func(agent *EinoReviewAgent) {
		if client != nil {
			agent.httpClient = client
		}
	}
}

func (a *EinoReviewAgent) Analyze(ctx context.Context, req AnalyzeRequest) (AnalyzeResult, error) {
	started := time.Now()
	graph, err := a.buildGraph(req)
	if err != nil {
		return AnalyzeResult{}, err
	}
	runnable, err := graph.Compile(ctx,
		compose.WithGraphName("eino_review_agent"),
		compose.WithMaxRunSteps(defaultGraphRunSteps),
	)
	if err != nil {
		return AnalyzeResult{}, fmt.Errorf("compile Eino review graph: %w", err)
	}

	state, err := runnable.Invoke(ctx, nodes.State{
		PullRequest:  req.PullRequest,
		RuleFindings: append([]review.Finding(nil), req.RuleFindings...),
	})
	if err != nil {
		return AnalyzeResult{}, err
	}
	state.Metrics.DurationMillis = time.Since(started).Milliseconds()
	if err := emit(ctx, req.EventSink, EventReviewCompleted, "review completed"); err != nil {
		return AnalyzeResult{}, err
	}
	return AnalyzeResult{
		Report:  state.FinalReport,
		Metrics: state.Metrics,
		Context: state,
	}, nil
}

func (a *EinoReviewAgent) buildGraph(req AnalyzeRequest) (*compose.Graph[nodes.State, nodes.State], error) {
	graph := compose.NewGraph[nodes.State, nodes.State]()

	build := nodes.ContextBuildNode{Config: req.Config}
	retrieve := nodes.ContextRetrieveNode{}
	compress := nodes.ContextCompressNode{Config: req.Config}
	prompt := nodes.PromptBuildNode{Config: req.Config}
	chat := nodes.ChatModelNode{Config: req.Config, HTTPClient: a.httpClient}
	parse := nodes.JSONParseNode{}
	validate := nodes.FindingValidateNode{Validate: Validator{}.ValidateReport}
	merge := nodes.MergeNode{}

	if err := addNode(graph, nodeContextBuild, req.EventSink, EventContextBuildStarted, EventContextBuildCompleted, build.Run); err != nil {
		return nil, err
	}
	if err := addNode(graph, nodeContextRetrieve, req.EventSink, "", EventContextRetrieveComplete, retrieve.Run); err != nil {
		return nil, err
	}
	if err := addNode(graph, nodeContextCompress, req.EventSink, "", EventContextCompressComplete, compress.Run); err != nil {
		return nil, err
	}
	if err := addNode(graph, nodePromptBuild, req.EventSink, "", "", prompt.Run); err != nil {
		return nil, err
	}
	if err := addNode(graph, nodeChatModel, req.EventSink, EventLLMCallStarted, EventLLMCallCompleted, chat.Run); err != nil {
		return nil, err
	}
	if err := addNode(graph, nodeJSONParse, req.EventSink, "", EventParseCompleted, parse.Run); err != nil {
		return nil, err
	}
	if err := addNode(graph, nodeFindingValidate, req.EventSink, "", EventValidateCompleted, validate.Run); err != nil {
		return nil, err
	}
	if err := addNode(graph, nodeMerge, req.EventSink, "", EventMergeCompleted, merge.Run); err != nil {
		return nil, err
	}

	return graph, addEdges(graph, []string{
		compose.START,
		nodeContextBuild,
		nodeContextRetrieve,
		nodeContextCompress,
		nodePromptBuild,
		nodeChatModel,
		nodeJSONParse,
		nodeFindingValidate,
		nodeMerge,
		compose.END,
	})
}

func addNode(
	graph *compose.Graph[nodes.State, nodes.State],
	key string,
	sink EventSink,
	startEvent string,
	completeEvent string,
	run func(context.Context, nodes.State) (nodes.State, error),
) error {
	wrapped := func(ctx context.Context, state nodes.State) (nodes.State, error) {
		if startEvent != "" {
			if err := emit(ctx, sink, startEvent, key+" started"); err != nil {
				return State{}, err
			}
		}
		next, err := run(ctx, state)
		if err != nil {
			return State{}, err
		}
		if completeEvent != "" {
			if err := emit(ctx, sink, completeEvent, key+" completed"); err != nil {
				return State{}, err
			}
		}
		return next, nil
	}
	if err := graph.AddLambdaNode(key, compose.InvokableLambda(wrapped)); err != nil {
		return fmt.Errorf("add Eino node %s: %w", key, err)
	}
	return nil
}

func addEdges(graph *compose.Graph[nodes.State, nodes.State], nodes []string) error {
	for i := 0; i < len(nodes)-1; i++ {
		if err := graph.AddEdge(nodes[i], nodes[i+1]); err != nil {
			return fmt.Errorf("add Eino edge %s -> %s: %w", nodes[i], nodes[i+1], err)
		}
	}
	return nil
}
