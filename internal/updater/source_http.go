package updater

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

func fetchReleaseAsset(ctx context.Context, c *http.Client, r Release, dst io.Writer, token string) error {
	if len(r.Assets) != 1 {
		return fmt.Errorf("Fetch requires release with exactly one selected asset")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.Assets[0].URL, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("asset status %d", resp.StatusCode)
	}
	_, err = io.Copy(dst, resp.Body)
	return err
}
