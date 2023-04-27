package app

import pixiv "github.com/littleneko/pixiv-api-go"

type pixivPageClient struct {
	client    *pixiv.PixivClient
	uid       string
	limit     int32
	total     int32
	curOffset int32
}

func (pc *pixivPageClient) CurOffset() int32 {
	return pc.curOffset
}

func (pc *pixivPageClient) Total() int32 {
	return pc.total
}

func (pc *pixivPageClient) MoveToNextPage() {
	pc.curOffset += pc.limit
}

func (pc *pixivPageClient) HasMorePage() bool {
	return pc.total == -1 || pc.curOffset < pc.total
}

// PixivBookmarksPageClient is a wrapper of PixivClient.GetUserBookmarks, it records the bookmarks page offset and num
type PixivBookmarksPageClient struct {
	pixivPageClient
}

func NewBookmarksPageClient(client *pixiv.PixivClient, uid string, limit int32) *PixivBookmarksPageClient {
	return &PixivBookmarksPageClient{
		pixivPageClient: pixivPageClient{
			client:    client,
			uid:       uid,
			limit:     limit,
			total:     -1,
			curOffset: 0,
		},
	}
}

func (bpc *PixivBookmarksPageClient) GetNextPageBookmarks() (*pixiv.BookmarksInfo, error) {
	bmInfo, err := bpc.client.GetUserBookmarks(bpc.uid, bpc.curOffset, bpc.limit)
	// mark this user as invalid user, it has no next page
	if err == pixiv.ErrNotFound {
		bpc.total = 0
	}
	if err != nil {
		return nil, err
	}
	if bmInfo.Total > bpc.total {
		bpc.total = bmInfo.Total
	}
	return bmInfo, nil
}

// PixivFollowingPageClient is a wrapper of PixivClient.GetUserFollowing, it records the bookmarks page offset and num
type PixivFollowingPageClient struct {
	pixivPageClient
}

func NewFollowingPageClient(client *pixiv.PixivClient, uid string, limit int32) *PixivFollowingPageClient {
	return &PixivFollowingPageClient{
		pixivPageClient: pixivPageClient{
			client:    client,
			uid:       uid,
			limit:     limit,
			total:     -1,
			curOffset: 0,
		},
	}
}

func (fpc *PixivFollowingPageClient) GetNextPageFollowing() (*pixiv.FollowingInfo, error) {
	bmInfo, err := fpc.client.GetUserFollowing(fpc.uid, fpc.curOffset, fpc.limit)
	// mark this user as invalid user, it has no next page
	if err == pixiv.ErrNotFound {
		fpc.total = 0
	}
	if err != nil {
		return nil, err
	}
	if bmInfo.Total > fpc.total {
		fpc.total = bmInfo.Total
	}
	return bmInfo, nil
}
