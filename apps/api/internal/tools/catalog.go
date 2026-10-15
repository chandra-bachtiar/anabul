package tools

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"anabul/internal/store"
)

func (s *Service) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.q.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Category, 0, len(rows))
	for _, row := range rows {
		result = append(result, Category{ID: row.ID, Name: row.Name, Keywords: row.Keywords})
	}
	return result, nil
}

// CategoryNames feeds the "did you mean" list in error messages.
func (s *Service) CategoryNames(ctx context.Context) ([]string, error) {
	rows, err := s.q.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.Name)
	}
	return names, nil
}

func (s *Service) CreateCategory(ctx context.Context, name string, keywords []string) (Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Category{}, invalid("nama kategori belum diisi")
	}
	if keywords == nil {
		keywords = []string{}
	}
	row, err := s.q.CreateCategory(ctx, store.CreateCategoryParams{Name: name, Keywords: keywords})
	if err != nil {
		if isUniqueViolation(err) {
			return Category{}, &Error{Code: CodeConflict, Message: "kategori " + name + " sudah ada"}
		}
		return Category{}, err
	}
	return Category{ID: row.ID, Name: row.Name, Keywords: row.Keywords}, nil
}

func (s *Service) UpdateCategory(ctx context.Context, id int64, name *string, keywords *[]string) (Category, error) {
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return Category{}, invalid("nama kategori tidak boleh kosong")
		}
		name = &trimmed
	}
	row, err := s.q.UpdateCategory(ctx, store.UpdateCategoryParams{
		ID:       id,
		Name:     textPtr(name),
		Keywords: derefStrings(keywords),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Category{}, notFound("kategori tidak ditemukan")
		}
		if isUniqueViolation(err) {
			return Category{}, &Error{Code: CodeConflict, Message: "nama kategori sudah dipakai"}
		}
		return Category{}, err
	}
	return Category{ID: row.ID, Name: row.Name, Keywords: row.Keywords}, nil
}

// DeleteCategory detaches the category from transactions, budgets, and
// recurring needs so none of that data is deleted along with it.
func (s *Service) DeleteCategory(ctx context.Context, id int64) error {
	category, err := s.getCategory(ctx, s.q, id)
	if err != nil {
		return err
	}
	return s.InTx(ctx, func(q *store.Queries) error {
		if err := q.DetachCategoryFromTransactions(ctx, int8Ptr(&category.ID)); err != nil {
			return err
		}
		if err := q.DeleteBudgetsForCategory(ctx, category.ID); err != nil {
			return err
		}
		if err := q.DetachCategoryFromRecurringNeeds(ctx, int8Ptr(&category.ID)); err != nil {
			return err
		}
		return q.DeleteCategory(ctx, category.ID)
	})
}

func (s *Service) ListAccounts(ctx context.Context) ([]Account, error) {
	rows, err := s.q.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Account, 0, len(rows))
	for _, row := range rows {
		result = append(result, Account{ID: row.ID, Name: row.Name})
	}
	return result, nil
}

func (s *Service) CreateAccount(ctx context.Context, name string) (Account, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Account{}, invalid("nama akun belum diisi")
	}
	row, err := s.q.CreateAccount(ctx, name)
	if err != nil {
		if isUniqueViolation(err) {
			return Account{}, &Error{Code: CodeConflict, Message: "akun " + name + " sudah ada"}
		}
		return Account{}, err
	}
	return Account{ID: row.ID, Name: row.Name}, nil
}

func (s *Service) getCategory(ctx context.Context, q *store.Queries, id int64) (*store.Category, error) {
	row, err := q.GetCategory(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFound("kategori tidak ditemukan")
		}
		return nil, err
	}
	return &row, nil
}

func (s *Service) getAccount(ctx context.Context, q *store.Queries, id int64) (*store.Account, error) {
	row, err := q.GetAccount(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, notFound("akun tidak ditemukan")
		}
		return nil, err
	}
	return &row, nil
}

// resolveCategory takes an id, a name, or a keyword guess from the note. Every
// path returns nil when nothing matches.
func (s *Service) resolveCategory(ctx context.Context, q *store.Queries, id *int64, name, note string) (*store.Category, error) {
	if id != nil {
		return s.getCategory(ctx, q, *id)
	}
	if strings.TrimSpace(name) != "" {
		row, err := q.FindCategoryByName(ctx, strings.TrimSpace(name))
		if err == nil {
			return &row, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		names, listErr := s.categoryNames(ctx, q)
		if listErr != nil {
			return nil, listErr
		}
		return nil, invalid("kategori %q tidak ada", name).WithAvailable(names...)
	}

	categories, err := q.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	candidates := make([]Category, 0, len(categories))
	for _, c := range categories {
		candidates = append(candidates, Category{ID: c.ID, Name: c.Name, Keywords: c.Keywords})
	}
	match := MatchCategory(note, candidates)
	if match == nil {
		return nil, nil
	}
	return s.getCategory(ctx, q, match.ID)
}

func (s *Service) resolveAccount(ctx context.Context, q *store.Queries, id *int64, name string) (*store.Account, error) {
	if id != nil {
		return s.getAccount(ctx, q, *id)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	row, err := q.FindAccountByName(ctx, name)
	if err == nil {
		return &row, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	accounts, listErr := q.ListAccounts(ctx)
	if listErr != nil {
		return nil, listErr
	}
	names := make([]string, 0, len(accounts))
	for _, a := range accounts {
		names = append(names, a.Name)
	}
	return nil, invalid("akun %q tidak ada", name).WithAvailable(names...)
}

func (s *Service) categoryNames(ctx context.Context, q *store.Queries) ([]string, error) {
	rows, err := q.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.Name)
	}
	return names, nil
}
