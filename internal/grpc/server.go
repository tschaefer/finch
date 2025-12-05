/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package grpc

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/tschaefer/finch/api"
	"github.com/tschaefer/finch/internal/config"
	"github.com/tschaefer/finch/internal/controller"
	"github.com/tschaefer/finch/internal/version"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AgentServer struct {
	api.UnimplementedAgentServiceServer
	controller controller.Controller
	config     config.Config
}

type InfoServer struct {
	api.UnimplementedInfoServiceServer
	config config.Config
}

func NewAgentServer(ctrl controller.Controller, cfg config.Config) *AgentServer {
	slog.Debug("Initializing gRPC AgentServer")
	return &AgentServer{
		controller: ctrl,
		config:     cfg,
	}
}

func NewInfoServer(cfg config.Config) *InfoServer {
	slog.Debug("Initializing gRPC InfoServer")
	return &InfoServer{
		config: cfg,
	}
}

func (s *AgentServer) RegisterAgent(ctx context.Context, req *api.RegisterAgentRequest) (*api.RegisterAgentResponse, error) {
	agent := &controller.Agent{
		Hostname:       req.Hostname,
		Tags:           req.Tags,
		LogSources:     req.LogSources,
		Metrics:        req.Metrics,
		MetricsTargets: req.MetricsTargets,
		Profiles:       req.Profiles,
	}

	rid, err := s.controller.RegisterAgent(agent)
	if err != nil {
		if errors.Is(err, controller.ErrAgentAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &api.RegisterAgentResponse{Rid: rid}, nil
}

func (s *AgentServer) DeregisterAgent(ctx context.Context, req *api.DeregisterAgentRequest) (*api.DeregisterAgentResponse, error) {
	err := s.controller.DeregisterAgent(req.Rid)
	if err != nil {
		if errors.Is(err, controller.ErrAgentNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &api.DeregisterAgentResponse{}, nil
}

func (s *AgentServer) GetAgent(ctx context.Context, req *api.GetAgentRequest) (*api.GetAgentResponse, error) {
	if req.Rid == "" {
		return nil, status.Error(codes.InvalidArgument, "resource ID is required")
	}

	agent, err := s.controller.GetAgent(req.Rid)
	if err != nil {
		if errors.Is(err, controller.ErrAgentNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &api.GetAgentResponse{
		ResourceId:     agent.ResourceId,
		Hostname:       agent.Hostname,
		Tags:           agent.Tags,
		LogSources:     agent.LogSources,
		Metrics:        agent.Metrics,
		MetricsTargets: agent.MetricsTargets,
		Profiles:       agent.Profiles,
		CreatedAt:      agent.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *AgentServer) ListAgents(ctx context.Context, req *api.ListAgentsRequest) (*api.ListAgentsResponse, error) {
	agentList, err := s.controller.ListAgents()
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	agents := make([]*api.AgentListItem, 0, len(agentList))
	for _, a := range agentList {
		agents = append(agents, &api.AgentListItem{
			Rid:      a["rid"],
			Hostname: a["hostname"],
		})
	}

	return &api.ListAgentsResponse{Agents: agents}, nil
}

func (s *AgentServer) GetAgentConfig(ctx context.Context, req *api.GetAgentConfigRequest) (*api.GetAgentConfigResponse, error) {
	if req.Rid == "" {
		return nil, status.Error(codes.InvalidArgument, "resource ID is required")
	}

	config, err := s.controller.CreateAgentConfig(req.Rid)
	if err != nil {
		if errors.Is(err, controller.ErrAgentNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &api.GetAgentConfigResponse{Config: config}, nil
}

func (s *InfoServer) GetServiceInfo(ctx context.Context, req *api.GetServiceInfoRequest) (*api.GetServiceInfoResponse, error) {
	return &api.GetServiceInfoResponse{
		Id:        s.config.Id(),
		Hostname:  s.config.Hostname(),
		CreatedAt: s.config.CreatedAt(),
		Release:   version.Release(),
		Commit:    version.Commit(),
	}, nil
}
