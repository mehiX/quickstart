import React, { useContext } from "react";

import Endpoint from "../Endpoint";
import Context from "../../Context";
import ProductTypesContainer from "./ProductTypesContainer";
import {
  transactionsCategories,
  authCategories,
  identityCategories,
  balanceCategories,
  investmentsCategories,
  paymentCategories,
  assetsCategories,
  transferCategories,
  transformAuthData,
  transformTransactionsData,
  transformBalanceData,
  transformInvestmentsData,
  transformIdentityData,
  transformPaymentData,
  transformAssetsData,
  transformTransferData,
} from "../../dataUtilities";

const Products = (props: any) => {
  const { products } = useContext(Context);
  return (
    <ProductTypesContainer productType="Products">
      {products.includes("payment_initiation") && (
        <Endpoint
          endpoint="payment"
          name="Payment"
          categories={paymentCategories}
          schema="/payment/get/"
          description="Retrieve information about your latest payment."
          transformData={transformPaymentData}
          userAuthToken={props.userAuthToken}
        />
      )}
      <Endpoint
        endpoint="auth"
        name="Auth"
        categories={authCategories}
        schema="/auth/get/"
        description="Retrieve account and routing numbers for checking and savings accounts."
        transformData={transformAuthData}
        userAuthToken={props.userAuthToken}
        />
      <Endpoint
        endpoint="transactions"
        name="Transactions"
        categories={transactionsCategories}
        schema="/transactions/get/"
        description="Retrieve transactions for credit and depository accounts."
        transformData={transformTransactionsData}
        userAuthToken={props.userAuthToken}
        />
      <Endpoint
        endpoint="identity"
        name="Identity"
        categories={identityCategories}
        schema="/identity/get/"
        description="Retrieve Identity information on file with the bank. Reduce
        fraud by comparing user-submitted data to validate identity."
        transformData={transformIdentityData}
        userAuthToken={props.userAuthToken}
        />
      {products.includes("assets") && (
        <Endpoint
          endpoint="assets"
          name="Assets"
          categories={assetsCategories}
          schema="/assets_report/get/"
          description="Create and retrieve assets information an asset report"
          transformData={transformAssetsData}
          userAuthToken={props.userAuthToken}
        />
      )}
      <Endpoint
        endpoint="balance"
        name="Balance"
        categories={balanceCategories}
        schema="/accounts/balance/get/"
        description="Check balances in real time to prevent non-sufficient funds
        fees."
        transformData={transformBalanceData}
        userAuthToken={props.userAuthToken}
        />
      <Endpoint
        endpoint="holdings"
        name="Investments"
        categories={investmentsCategories}
        schema="/investments/holdings/get/"
        description="Retrieve investment holdings on file with the bank,
        brokerage, or investment institution. Analyze over-exposure
        to market segments."
        transformData={transformInvestmentsData}
        userAuthToken={props.userAuthToken}
        />
      {products.includes("transfer") && (
        <Endpoint
          endpoint="transfer"
          name="Transfer"
          categories={transferCategories}
          schema="/transfer/get/"
          description="Retrieve information about your latest ACH Transfer."
          transformData={transformTransferData}
          userAuthToken={props.userAuthToken}
        />
      )}
    </ProductTypesContainer>
  );
};

Products.displayName = "Products";

export default Products;
