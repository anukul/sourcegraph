package graphqlbackend

import (
	"context"
	"sync"
	"time"

	"sourcegraph.com/sourcegraph/sourcegraph/pkg/backend"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/gitserver"
	"sourcegraph.com/sourcegraph/sourcegraph/pkg/gitserver/protocol"
)

func (r *repositoryResolver) MirrorInfo() *repositoryMirrorInfoResolver {
	return &repositoryMirrorInfoResolver{repository: r}
}

type repositoryMirrorInfoResolver struct {
	repository *repositoryResolver

	// memoize the gitserver RepoInfo call
	repoInfoOnce     sync.Once
	repoInfoResponse *protocol.RepoInfoResponse
	repoInfoErr      error
}

func (r *repositoryMirrorInfoResolver) gitserverRepoInfo(ctx context.Context) (*protocol.RepoInfoResponse, error) {
	r.repoInfoOnce.Do(func() {
		r.repoInfoResponse, r.repoInfoErr = gitserver.DefaultClient.RepoInfo(ctx, r.repository.repo.URI)
	})
	return r.repoInfoResponse, r.repoInfoErr
}

func (r *repositoryMirrorInfoResolver) RemoteURL(ctx context.Context) (string, error) {
	// 🚨 SECURITY: The remote URL might contain secret credentials in the URL userinfo, so
	// only allow site admins to see it.
	if err := backend.CheckCurrentUserIsSiteAdmin(ctx); err != nil {
		return "", err
	}

	info, err := r.gitserverRepoInfo(ctx)
	if err != nil {
		return "", err
	}
	return info.URL, nil
}

func (r *repositoryMirrorInfoResolver) Cloned(ctx context.Context) (bool, error) {
	info, err := r.gitserverRepoInfo(ctx)
	if err != nil {
		return false, err
	}
	return info.Cloned, nil
}

func (r *repositoryMirrorInfoResolver) CloneInProgress(ctx context.Context) (bool, error) {
	info, err := r.gitserverRepoInfo(ctx)
	if err != nil {
		return false, err
	}
	return info.CloneInProgress, nil
}

func (r *repositoryMirrorInfoResolver) UpdatedAt(ctx context.Context) (*string, error) {
	info, err := r.gitserverRepoInfo(ctx)
	if err != nil {
		return nil, err
	}
	if info.LastFetched == nil {
		return nil, err
	}
	s := info.LastFetched.Format(time.RFC3339)
	return &s, nil
}
