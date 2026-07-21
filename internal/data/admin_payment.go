package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/npanel-dev/NPanel-backend/ent"
	"github.com/npanel-dev/NPanel-backend/ent/proxypayment"
	paymentbiz "github.com/npanel-dev/NPanel-backend/internal/biz/admin/payment"
	"github.com/npanel-dev/NPanel-backend/internal/responsecode"
	paymentpkg "github.com/npanel-dev/NPanel-backend/pkg/payment"
	stripepkg "github.com/npanel-dev/NPanel-backend/pkg/payment/stripe"
	"github.com/npanel-dev/NPanel-backend/pkg/tool"
)

type adminPaymentRepo struct {
	data *Data
	log  *log.Helper
}

func NewAdminPaymentRepo(data *Data, logger log.Logger) paymentbiz.PaymentRepo {
	return &adminPaymentRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

func (r *adminPaymentRepo) Create(ctx context.Context, method *paymentbiz.PaymentMethod) (*paymentbiz.PaymentMethod, error) {
	token := tool.GenerateRandomString(8)
	config, err := r.normalizePaymentConfig(method.Platform, method.Config)
	if err != nil {
		return nil, err
	}

	sort := method.Sort
	if sort == 0 {
		sort, err = r.nextSort(ctx)
		if err != nil {
			return nil, err
		}
	}

	if paymentpkg.ParsePlatform(method.Platform) == paymentpkg.Stripe {
		config, err = r.attachStripeWebhook(config, strings.TrimSpace(method.Domain), token)
		if err != nil {
			return nil, err
		}
	}

	created, err := r.data.db.ProxyPayment.
		Create().
		SetName(method.Name).
		SetPlatform(method.Platform).
		SetDescription(method.Description).
		SetIcon(method.Icon).
		SetDomain(method.Domain).
		SetConfig(config).
		SetFeeMode(uint(method.FeeMode)).
		SetFeePercent(method.FeePercent).
		SetFeeAmount(method.FeeAmount).
		SetSort(int32(sort)).
		SetEnable(method.Enable).
		SetToken(token).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return r.toBiz(created), nil
}

func (r *adminPaymentRepo) Update(ctx context.Context, method *paymentbiz.PaymentMethod) (*paymentbiz.PaymentMethod, error) {
	original, err := r.data.db.ProxyPayment.
		Query().
		Where(proxypayment.ID(method.ID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, responsecode.NewPaymentNotFoundError()
		}
		return nil, err
	}

	config, err := r.normalizePaymentConfig(method.Platform, method.Config)
	if err != nil {
		return nil, err
	}

	sort := method.Sort
	if sort == 0 {
		sort = int64(original.Sort)
	}

	updated, err := original.
		Update().
		SetName(method.Name).
		SetPlatform(method.Platform).
		SetDescription(method.Description).
		SetIcon(method.Icon).
		SetDomain(method.Domain).
		SetConfig(config).
		SetFeeMode(uint(method.FeeMode)).
		SetFeePercent(method.FeePercent).
		SetFeeAmount(method.FeeAmount).
		SetSort(int32(sort)).
		SetEnable(method.Enable).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return r.toBiz(updated), nil
}

func (r *adminPaymentRepo) Delete(ctx context.Context, id int) error {
	_, err := r.data.db.ProxyPayment.
		Delete().
		Where(proxypayment.ID(int64(id))).
		Exec(ctx)
	return err
}

func (r *adminPaymentRepo) Get(ctx context.Context, id int) (*paymentbiz.PaymentMethod, error) {
	payment, err := r.data.db.ProxyPayment.
		Query().
		Where(proxypayment.ID(int64(id))).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, responsecode.NewPaymentNotFoundError()
		}
		return nil, err
	}
	return r.toBiz(payment), nil
}

func (r *adminPaymentRepo) List(ctx context.Context, page, size int, platform, search string, enable *bool) (int32, []*paymentbiz.PaymentMethod, error) {
	query := r.data.db.ProxyPayment.Query()
	if platform != "" {
		query = query.Where(proxypayment.Platform(platform))
	}
	if enable != nil {
		query = query.Where(proxypayment.Enable(*enable))
	}
	if search != "" {
		query = query.Where(proxypayment.NameContainsFold(search))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return 0, nil, err
	}

	payments, err := query.
		Offset((page-1)*size).
		Limit(size).
		Order(ent.Asc(proxypayment.FieldSort), ent.Asc(proxypayment.FieldID)).
		All(ctx)
	if err != nil {
		return 0, nil, err
	}

	result := make([]*paymentbiz.PaymentMethod, 0, len(payments))
	for _, item := range payments {
		result = append(result, r.toBiz(item))
	}
	return int32(total), result, nil
}

func (r *adminPaymentRepo) toBiz(item *ent.ProxyPayment) *paymentbiz.PaymentMethod {
	if item == nil {
		return nil
	}
	return &paymentbiz.PaymentMethod{
		ID:          item.ID,
		Name:        item.Name,
		Platform:    item.Platform,
		Description: item.Description,
		Icon:        item.Icon,
		Domain:      item.Domain,
		Config:      item.Config,
		FeeMode:     int32(item.FeeMode),
		FeePercent:  item.FeePercent,
		FeeAmount:   item.FeeAmount,
		Sort:        int64(item.Sort),
		Enable:      item.Enable,
		Token:       item.Token,
		SiteHost:    r.siteHost(),
	}
}

func (r *adminPaymentRepo) nextSort(ctx context.Context) (int64, error) {
	last, err := r.data.db.ProxyPayment.Query().Order(ent.Desc(proxypayment.FieldSort)).First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 1, nil
		}
		return 0, err
	}
	return int64(last.Sort) + 1, nil
}

func (r *adminPaymentRepo) normalizePaymentConfig(platform, raw string) (string, error) {
	data := []byte(raw)
	switch paymentpkg.ParsePlatform(platform) {
	case paymentpkg.Stripe:
		var cfg StripeConfig
		if err := cfg.Unmarshal(data); err != nil {
			return "", err
		}
		content, err := json.Marshal(&cfg)
		if err != nil {
			return "", err
		}
		return string(content), nil
	case paymentpkg.AlipayF2F:
		var cfg AlipayF2FConfig
		if err := cfg.Unmarshal(data); err != nil {
			return "", err
		}
		content, err := json.Marshal(&cfg)
		if err != nil {
			return "", err
		}
		return string(content), nil
	case paymentpkg.EPay:
		var cfg EPayConfig
		if err := cfg.Unmarshal(data); err != nil {
			return "", err
		}
		content, err := json.Marshal(&cfg)
		if err != nil {
			return "", err
		}
		return string(content), nil
	case paymentpkg.CryptoSaaS:
		var cfg CryptoSaaSConfig
		if err := cfg.Unmarshal(data); err != nil {
			return "", err
		}
		content, err := json.Marshal(&cfg)
		if err != nil {
			return "", err
		}
		return string(content), nil
	default:
		return raw, nil
	}
}

func (r *adminPaymentRepo) attachStripeWebhook(configJSON, domain, token string) (string, error) {
	var cfg StripeConfig
	if err := cfg.Unmarshal([]byte(configJSON)); err != nil {
		return "", err
	}
	if strings.TrimSpace(cfg.SecretKey) == "" {
		return "", fmt.Errorf("stripe secret key is empty")
	}

	client := stripepkg.NewClient(stripepkg.Config{
		SecretKey: cfg.SecretKey,
		PublicKey: cfg.PublicKey,
	})
	endpoint, err := client.CreateWebhookEndpoint(fmt.Sprintf("%s/v1/notify/Stripe/%s", domain, token))
	if err != nil {
		return "", err
	}
	cfg.WebhookSecret = endpoint.Secret

	content, err := json.Marshal(&cfg)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (r *adminPaymentRepo) siteHost() string {
	if r.data != nil && r.data.AppConf() != nil && r.data.AppConf().Site != nil {
		return strings.TrimSpace(r.data.AppConf().Site.Host)
	}
	return ""
}
